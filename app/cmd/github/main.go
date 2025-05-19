package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hupe1980/tokenbridge"
	"github.com/hupe1980/tokenbridge-backend/internal"
)

var (
	kmsClient *kms.Client
	keyID     string
)

func init() {
	// Load AWS SDK configuration
	slog.Info("Initializing AWS SDK configuration...")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		slog.Error("Failed to load AWS SDK configuration", "error", err)
		os.Exit(1)
	}

	// Initialize the KMS client
	slog.Info("Initializing KMS client...")
	kmsClient = kms.NewFromConfig(cfg)

	// Fetch the KMS_KEY_ID environment variable
	keyID = os.Getenv("KMS_KEY_ID")
	if keyID == "" {
		slog.Error("KMS_KEY_ID environment variable is not set")
		os.Exit(1)
	}
	slog.Info("KMS_KEY_ID loaded successfully")
}

func handleRequest(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	logger := slog.With("request_id", event.RequestContext.RequestID)
	logger.Info("Handling new request")

	// Decode x-www-form-urlencoded payload
	values, err := internal.ParseBody(event)
	if err != nil {
		logger.Error("Failed to parse form payload", "error", err)
		return internal.ErrorResponse(400, "failed to parse form payload"), nil
	}

	subjectToken := values.Get("subject_token")

	// customAttributesStr := values.Get("custom_attributes")

	// var customAttributes map[string]any
	// if customAttributesStr != "" {
	// 	if err := json.Unmarshal([]byte(customAttributesStr), &customAttributes); err != nil {
	// 		logger.Error("Failed to decode custom_attributes JSON", "error", err)
	// 		return internal.ErrorResponse(400, "failed to decode custom_attributes JSON"), nil
	// 	}
	// } else {
	// 	customAttributes = make(map[string]any)
	// }

	// Validate subject_token
	if subjectToken == "" {
		logger.Warn("subject_token is missing in the payload")
		return internal.ErrorResponse(400, "subject_token is missing"), nil
	}

	// Create OIDC verifier
	// Parse issuer URL
	issuerURL, err := url.Parse("https://token.actions.githubusercontent.com")
	if err != nil {
		logger.Error("Failed to parse issuer URL", "error", err)
		return internal.ErrorResponse(500, "failed to parse issuer URL"), nil
	}

	oidcVerifier, err := tokenbridge.NewOIDCVerifier(ctx, issuerURL, []string{"tokenbridge"}, func(o *tokenbridge.OIDCVerifierOptions) {
		o.Thumbprints = []string{"D89E3BD43D5D909B47A18977AA9D5CE36CEE184C"}
	})
	if err != nil {
		logger.Error("Failed to create OIDC verifier", "error", err)
		return internal.ErrorResponse(500, "failed to create OIDC verifier"), nil
	}

	// Initialize RSA signer
	rsaSigner := tokenbridge.NewKMSSigner(kmsClient, keyID, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256)

	// Construct full URL
	fullURL := &url.URL{
		Scheme: "https",
		Host:   event.RequestContext.DomainName,
	}

	// Create Issuer
	issuer := tokenbridge.NewTokenIssuerWithJWKS(fullURL.String(), rsaSigner, func(o *tokenbridge.TokenIssuerWithJWKSOptions) {
		o.OnTokenCreate = func(ctx context.Context, issuer string, idToken *oidc.IDToken) (jwt.MapClaims, error) {
			claims, err := tokenbridge.DefaultOnTokenCreate(ctx, issuer, idToken)
			if err != nil {
				logger.Error("Failed to create token claims", "error", err)
				return nil, err
			}

			// Add the request ID as the JWT ID (jti) claim
			claims["jti"] = event.RequestContext.RequestID

			return claims, nil
		}
	})

	// Create TokenBridge
	tb := tokenbridge.New(oidcVerifier, issuer)

	// Exchange token
	result, err := tb.ExchangeToken(ctx, subjectToken)
	if err != nil {
		logger.Error("Token exchange failed", "error", err)
		return internal.ErrorResponse(401, "token exchange failed"), nil
	}

	logger.Info("Token exchange successful")
	respBody, _ := json.Marshal(result)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(respBody),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": "no-store",
		},
	}, nil
}

func main() {
	slog.Info("Starting Lambda function...")
	lambda.Start(handleRequest)
}
