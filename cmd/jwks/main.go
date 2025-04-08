package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
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

func handleRequest(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	logger := slog.With("request_id", event.RequestContext.RequestID)
	logger.Info("Handling new request")

	// Initialize RSA signer
	rsaSigner := signer.NewKMS(kmsClient, keyID, types.SigningAlgorithmSpecRsassaPkcs1V15Sha256)

	// Construct full URL
	fullURL := &url.URL{
		Scheme: "https",
		Host:   event.RequestContext.DomainName,
	}
	logger.Info("Constructed full URL", "fullURL", fullURL.String())

	// Create AuthServer
	authServer := tokenbridge.NewAuthServer(fullURL.String(), rsaSigner)

	// Get JWKS
	logger.Info("Fetching JWKS...")
	jwks, err := authServer.GetJWKS(ctx)
	if err != nil {
		logger.Error("Failed to fetch JWKS", "error", err)
		return errorResponse(500, "failed to get JWKS"), nil
	}
	logger.Info("Successfully fetched JWKS")

	// Marshal JWKS response
	respBody, _ := json.Marshal(jwks)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(respBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func errorResponse(status int, msg string) events.APIGatewayV2HTTPResponse {
	body := fmt.Sprintf(`{"error":"%s"}`, msg)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       body,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

func main() {
	slog.Info("Starting Lambda function...")
	lambda.Start(handleRequest)
}
