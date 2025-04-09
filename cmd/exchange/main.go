package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hupe1980/tokenbridge"
	"github.com/hupe1980/tokenbridge/signer"
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

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func handleRequest(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	logger := slog.With("request_id", event.RequestContext.RequestID)
	logger.Info("Handling new request")

	// Parse issuer URL
	issuerURL, err := url.Parse("https://token.actions.githubusercontent.com")
	if err != nil {
		logger.Error("Failed to parse issuer URL", "error", err)
		return errorResponse(500, "failed to parse issuer URL"), nil
	}

	// Decode JSON payload
	var payload struct {
		IDToken      string         `json:"id_token"`
		CustomClaims map[string]any `json:"custom_claims"`
	}
	if err := json.NewDecoder(strings.NewReader(event.Body)).Decode(&payload); err != nil {
		logger.Error("Failed to decode JSON payload", "error", err)
		return errorResponse(400, "failed to decode JSON payload"), nil
	}

	// Validate ID token
	if payload.IDToken == "" {
		logger.Warn("ID token is missing in the payload")
		return errorResponse(400, "ID token is missing"), nil
	}

	// Create OIDC verifier
	oidcVerifier, err := tokenbridge.NewOIDCVerifier(ctx, issuerURL, []string{"tokenbridge"}, func(o *tokenbridge.OIDCVerifierOptions) {
		o.Thumbprints = []string{"D89E3BD43D5D909B47A18977AA9D5CE36CEE184C"}
	})
	if err != nil {
		logger.Error("Failed to create OIDC verifier", "error", err)
		return errorResponse(500, "failed to create OIDC verifier"), nil
	}

	// Initialize RSA signer
	rsaSigner := signer.NewKMS(kmsClient, keyID, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256)

	// Construct full URL
	fullURL := &url.URL{
		Scheme: "https",
		Host:   event.RequestContext.DomainName,
	}

	// Create AuthServer and TokenBridge
	authServer := tokenbridge.NewAuthServer(fullURL.String(), rsaSigner, func(o *tokenbridge.AuthServerOptions) {
		o.OnTokenCreate = func(ctx context.Context, idToken *oidc.IDToken) (jwt.MapClaims, error) {
			claims := jwt.MapClaims{
				"iss": idToken.Issuer,
				"sub": idToken.Subject,
				"aud": idToken.Audience,
			}

			// Add custom claims if provided
			for key, value := range payload.CustomClaims {
				if key != "" {
					if slices.Contains([]string{"iss", "sub", "aud"}, key) {
						logger.Warn("Attempt to overwrite reserved claim", "claim", key)
						return nil, fmt.Errorf("custom claim '%s' cannot overwrite reserved claims", key)
					}
					claims[key] = value
				}
			}
			return claims, nil
		}
	})
	tb := tokenbridge.New(oidcVerifier, authServer)

	// Exchange token
	accessToken, err := tb.ExchangeToken(ctx, payload.IDToken)
	if err != nil {
		logger.Error("Token exchange failed", "error", err)
		return errorResponse(401, "token exchange failed"), nil
	}

	logger.Info("Token exchange successful")
	respBody, _ := json.Marshal(AccessTokenResponse{AccessToken: accessToken})

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(respBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func errorResponse(status int, msg string) events.APIGatewayV2HTTPResponse {
	body, _ := json.Marshal(map[string]string{"error": msg})
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

func main() {
	slog.Info("Starting Lambda function...")
	lambda.Start(handleRequest)
}
