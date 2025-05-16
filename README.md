# TokenBridge Backend Example

This repository contains the backend implementation for the TokenBridge service. It includes AWS Lambda functions written in Go, an API Gateway (HTTP API), and integration with AWS KMS for secure token signing and verification.

## Features

- **AWS Lambda Functions**: Backend logic implemented in Go.
- **API Gateway (HTTP API)**: Provides endpoints for token exchange and JWKS retrieval.
- **AWS KMS Integration**: Uses an asymmetric RSA key for signing and verifying tokens.
- **Structured Logging**: Uses `slog` for structured and traceable logs.

## Endpoints

### `/exchange` (POST)
- **Description**: Exchanges a subject token for an access token.
- **Request Content-Type**: `application/x-www-form-urlencoded`
- **Request Body**:
  ```
  subject_token=<SUBJECT_TOKEN>&custom_attributes=<JSON_OBJECT>
  ```
  - `subject_token` (required): The token to be exchanged.
  - `custom_attributes` (optional): A JSON string of additional claims to include in the access token. Example: `{"foo":"bar"}`

- **Response**:
  ```json
  {
    "access_token": "<ACCESS_TOKEN>",
    "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
    "token_type": "Bearer"
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

## Environment Variables

- `KMS_KEY_ID`: The AWS KMS key ID used for signing tokens.

## Related Projects

- [**TokenBridge**](https://github.com/hupe1980/tokenbridge): The main project for TokenBridge, providing core functionality and documentation.
- [**TokenBridge GitHub Action**](https://github.com/hupe1980/tokenbridge-action): Automate your workflows with TokenBridge using GitHub Actions.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.