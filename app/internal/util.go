package internal

import (
	"encoding/base64"
	"encoding/json"
	"net/url"

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

// ParseBody parses the request body, handling base64 encoding and form decoding.
func ParseBody(event events.APIGatewayV2HTTPRequest) (url.Values, error) {
	body := event.Body
	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return nil, err
		}

		body = string(decoded)
	}

	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, err
	}

	return values, nil
}
