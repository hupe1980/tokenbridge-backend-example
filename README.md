# TokenBridge Backend Example

This repository contains the backend implementation for the TokenBridge service. It includes AWS Lambda functions written in Go, an API Gateway (HTTP API), and integration with AWS KMS for secure token signing and verification.

## Features

- **AWS Lambda Functions**: Backend logic implemented in Go.
- **API Gateway (HTTP API)**: Provides endpoints for token exchange and JWKS retrieval.
- **AWS KMS Integration**: Uses an asymmetric RSA key for signing and verifying tokens.
- **Structured Logging**: Uses `slog` for structured and traceable logs.

## Endpoints

### `/exchange` (POST)
- **Description**: Exchanges an ID token for an access token.
- **Request Body**:
  ```json
  {
    "id_token": "<ID_TOKEN>",
    "custom_claims": {
      "key": "value"
    }
  }
  ```
- **Response**:
  ```json
  {
    "access_token": "<ACCESS_TOKEN>"
  }
  ```

### `/.well-known/jwks.json` (GET)
- **Description**: Retrieves the JSON Web Key Set (JWKS) for token verification.
- **Response**:
  ```json
  {
    "keys": [
      {
        "kty": "RSA",
        "kid": "<KEY_ID>",
        "use": "sig",
        "alg": "RS256",
        "n": "<MODULUS>",
        "e": "<EXPONENT>"
      }
    ]
  }
  ```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.