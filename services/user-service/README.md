# User Service

User authentication and management service for AIPX platform.

## Features

- **User Authentication**
  - Signup with email and password
  - Login with email and password
  - JWT-based authentication (access + refresh tokens)
  - Secure password hashing with Argon2id
  - Token refresh and logout

- **User Management**
  - Get user profile
  - Update user profile
  - Delete user account (soft delete)

- **Security**
  - Argon2id password hashing (OWASP recommended parameters)
  - JWT tokens with RS256 signing
  - AES-256-GCM encryption for API keys
  - Secure token storage with hashing
  - Request logging and audit trails

## API Endpoints

### Public Endpoints

- `POST /api/v1/auth/signup` - Register new user
- `POST /api/v1/auth/login` - Login with credentials
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout and invalidate token
- `GET /health` - Health check

### Protected Endpoints (Requires Authentication)

- `GET /api/v1/users/me` - Get current user profile
- `PATCH /api/v1/users/me` - Update current user profile
- `DELETE /api/v1/users/me` - Delete current user account

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

### Environment Variables

- **Server**: `SERVER_HOST`, `SERVER_PORT`, `ENVIRONMENT`
- **Database**: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- **JWT**: `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`
- **Encryption**: `ENCRYPTION_MASTER_KEY`
- **Logging**: `LOG_LEVEL`, `LOG_FORMAT`

### Generate Secure Secrets

```bash
# Generate JWT secrets
openssl rand -base64 64

# Generate encryption key
openssl rand -base64 32
```

## Build and Run

### Build

```bash
go mod tidy
go build -o bin/user-service ./cmd/server
```

### Run

```bash
./bin/user-service
```

## Docker

```bash
# Build image
docker build -t aipx/user-service:latest .

# Run container
docker run -p 8080:8080 --env-file .env aipx/user-service:latest
```

## API Examples

### Signup

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123",
    "name": "John Doe"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123"
  }'
```

### Get User Profile

```bash
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

### Update Profile

```bash
curl -X PATCH http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Jane Doe"
  }'
```

## Response Format

### Success Response

```json
{
  "success": true,
  "data": { ... },
  "message": "Operation successful",
  "timestamp": "2025-11-19T10:00:00Z"
}
```

### Error Response

```json
{
  "success": false,
  "error": "Error message",
  "details": ["Detail 1", "Detail 2"],
  "timestamp": "2025-11-19T10:00:00Z"
}
```

## Security Features

- **Password Security**: Argon2id with 64MB memory, 3 iterations
- **Token Security**: JWT with HMAC-SHA256, 15min access, 7day refresh
- **API Key Encryption**: AES-256-GCM with authenticated encryption
- **Audit Logging**: All authentication events tracked
- **Rate Limiting**: (To be implemented with Redis)

## Architecture

```
cmd/server/main.go          # HTTP server entry point
internal/
  ├── auth/
  │   ├── jwt.go            # JWT token management
  │   └── password.go       # Password hashing (Argon2id)
  ├── handlers/
  │   ├── auth_handler.go   # Authentication endpoints
  │   └── user_handler.go   # User management endpoints
  ├── middleware/
  │   ├── auth.go           # JWT authentication
  │   ├── cors.go           # CORS middleware
  │   ├── logger.go         # Request logging
  │   └── recovery.go       # Panic recovery
  ├── models/
  │   ├── models.go         # Domain models
  │   ├── request.go        # API request DTOs
  │   └── response.go       # API response DTOs
  ├── repository/
  │   ├── user_repo.go      # User data access
  │   └── token_repo.go     # Token data access
  ├── config/
  │   └── config.go         # Configuration management
  └── crypto/
      └── aes.go            # AES encryption utilities
```

## Development

### Run Tests

```bash
go test ./...
```

### Run with Hot Reload

```bash
air
```

## License

Proprietary - AIPX Platform
