package internal

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

// ErrorResponse generates a structured HTTP error response for API Gateway (HTTP API).
func ErrorResponse(status int, msg string) events.APIGatewayV2HTTPResponse {
	body, _ := json.Marshal(map[string]string{"error": msg})
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}
