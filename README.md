# Authentication Service

Authentication service for Lee-Tech platform, built using the core package for production-ready microservice functionality.

## Features

- **User Registration**: Create new user accounts with email and username
- **User Login**: Authenticate users with username/password
- **Dynamic Organization Model**: Model organizations, departments, and user memberships with per-tenant hierarchy support
- **JWT Token Management**: Issue and validate access/refresh tokens enriched with organization roles and memberships
- **Account Security**: Password hashing with bcrypt, login attempt tracking, account lockout
- **Administrative APIs**: Super-admin endpoints to manage organizations, departments, and user assignments
- **Health Checks**: Comprehensive health endpoints for monitoring
- **Core Integration**: Leverages core package for middleware, error handling, logging, and more

## API Endpoints

### Authentication Endpoints

#### 1. Register User
```bash
POST /api/v1/authentication/register

Request Body:
{
  "email": "user@example.com",
  "username": "johndoe",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe"
}

Response (201):
{
  "message": "User registered successfully",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "johndoe",
    "first_name": "John",
    "last_name": "Doe",
    "primary_organization_id": null,
    "primary_department_id": null,
    "is_super_admin": false,
    "mfa_enabled": false,
    "organizations": [],
    "departments": []
  }
}
```

#### 2. Login
```bash
POST /api/v1/authentication/login

Request Body:
{
  "username": "johndoe",  // Can be email or username
  "password": "SecurePass123!"
}

Response (200):
{
  "access_token": "eyJhbGciOiJIUzI1...",
  "refresh_token": "eyJhbGciOiJIUzI1...",
  "expires_in": 900,  // seconds
  "token_type": "Bearer",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "johndoe",
    "first_name": "John",
    "last_name": "Doe",
    "primary_organization_id": "a3dc9340-9f20-4c47-a9f6-2a6628fd5d1d",
    "primary_department_id": "f5d4db52-69ed-4607-a0ec-8e1d20db2518",
    "is_super_admin": false,
    "mfa_enabled": false,
    "organizations": [
      {
        "organization_id": "a3dc9340-9f20-4c47-a9f6-2a6628fd5d1d",
        "organization_name": "Lee Tech HQ",
        "role": "CEO",
        "is_primary": true
      }
    ],
    "departments": [
      {
        "department_id": "f5d4db52-69ed-4607-a0ec-8e1d20db2518",
        "department_name": "Phong Kinh Doanh",
        "role": "LEAD",
        "is_primary": true
      }
    ]
  }
}
```

#### 3. Refresh Token
```bash
POST /api/v1/authentication/refresh

Request Body:
{
  "refresh_token": "eyJhbGciOiJIUzI1..."
}

Response (200):
{
  "access_token": "eyJhbGciOiJIUzI1...",
  "refresh_token": "eyJhbGciOiJIUzI1...",
  "expires_in": 900,
  "token_type": "Bearer",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "johndoe",
    "first_name": "John",
    "last_name": "Doe",
    "primary_organization_id": "a3dc9340-9f20-4c47-a9f6-2a6628fd5d1d",
    "primary_department_id": "f5d4db52-69ed-4607-a0ec-8e1d20db2518",
    "is_super_admin": false,
    "mfa_enabled": false,
    "organizations": [...],
    "departments": [...]
  }
}
```

The issued access token now includes organization context for ABAC-aware services:

```json
{
  "sub": "0cb0a1b2-7f1d-4b43-b43d-a4fcbf7e0140",
  "email": "johndoe@example.com",
  "roles": ["CEO", "FIELD_MANAGER"],
  "organizations": [
    {"id": "a3dc9340-9f20-4c47-a9f6-2a6628fd5d1d", "name": "Lee Tech HQ", "role": "CEO", "is_primary": true},
    {"id": "9bcb58c0-0497-4a65-8b67-0c1d8d4e8235", "name": "Lee Tech South", "role": "DIRECTOR", "is_primary": false}
  ],
  "departments": [
    {"id": "f5d4db52-69ed-4607-a0ec-8e1d20db2518", "name": "Phong Kinh Doanh", "role": "LEAD", "is_primary": true}
  ]
}
```

### Health Check Endpoints

```bash
GET /api/v1/authentication/health    # Basic health check
GET /healthz                # Kubernetes liveness probe
GET /health                 # Simple health status
GET /live                   # Liveness check
GET /ready                  # Readiness check
GET /health/detailed        # Detailed health with all checks
```

### Authenticated User Endpoint

```bash
GET /api/v1/authentication/me
Authorization: Bearer <access token>
```

Returns the same `user` projection as the login response, ensuring clients can refresh membership information after assignment changes.

### Administrative Endpoints (Super Admin)

The following routes require super-admin access and are intended for tenant bootstrapping and org chart maintenance:

| Method | Path | Description |
| ------ | ---- | ----------- |
| `POST` | `/api/v1/authentication/admin/organizations` | Create a new organization/tenant or child unit |
| `GET`  | `/api/v1/authentication/admin/organizations` | List organizations (includes hierarchical relationships) |
| `POST` | `/api/v1/authentication/admin/organizations/{organization_id}/departments` | Create a department/division under an organization |
| `GET`  | `/api/v1/authentication/admin/organizations/{organization_id}/departments` | List departments within an organization |
| `POST` | `/api/v1/authentication/admin/organizations/{organization_id}/members` | Assign a user to an organization, optionally as primary |
| `POST` | `/api/v1/authentication/admin/departments/{department_id}/members` | Assign a user to a department/team |
| `GET`  | `/api/v1/authentication/admin/users` | Paginated list of users (requires `auth.users.read` or super admin) |
| `GET`  | `/api/v1/authentication/admin/users/{user_id}/organizations` | List a user's organization memberships |
| `GET`  | `/api/v1/authentication/admin/users/{user_id}/departments` | List a user's department memberships |

#### Example: Create Department

```bash
POST /api/v1/authentication/admin/organizations/{org_id}/departments
Authorization: Bearer <super-admin token>
Content-Type: application/json

{
  "parent_id": null,
  "code": "BUSINESS",
  "name": "Phong Kinh Doanh",
  "kind": "DEPARTMENT",
  "description": "Phat trien khach hang va quan ly doanh thu.",
  "function": "Phat trien va duy tri doanh thu."
}
```

#### Example: Assign User to Organization

```bash
POST /api/v1/authentication/admin/organizations/{org_id}/members
Authorization: Bearer <super-admin token>
Content-Type: application/json

{
  "user_id": "0cb0a1b2-7f1d-4b43-b43d-a4fcbf7e0140",
  "role": "CEO",
  "is_primary": true
}
```

Successful calls immediately affect `/me` and login responses, ensuring dependent services receive up-to-date claims.

## Configuration

The service uses environment variables for configuration. Copy `.env.example` to `.env` and adjust as needed:

```bash
cp .env.example .env
```

### Bootstrap Defaults

On startup, the service ensures a root organization and super-administrator user exist. Override the defaults with these environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `BOOTSTRAP_ORG_NAME` | `Root Organization` | Display name for the seed organization |
| `BOOTSTRAP_ORG_DESCRIPTION` | `System root organization` | Optional description |
| `BOOTSTRAP_ORG_DOMAIN` | `root.local` | Domain/slug used to look up the organization |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@root.local` | Login email for the bootstrap admin |
| `BOOTSTRAP_ADMIN_USERNAME` | `root-admin` | Username for the bootstrap admin |
| `BOOTSTRAP_ADMIN_PASSWORD` | `ChangeMe123!` | Initial password (must meet `PASSWORD_MIN_LENGTH`) |
| `BOOTSTRAP_ADMIN_FIRST_NAME` | `System` | Admin first name |
| `BOOTSTRAP_ADMIN_LAST_NAME` | `Administrator` | Admin last name |

Change the password immediately after the first login. Set `DISABLE_AUTHORIZATION=true` if you need to run the service without contacting the authorization API during bootstrap.

Run the one-off bootstrap utility to rotate credentials without changing environment variables:

```bash
go run ./cmd/bootstrap \
  --admin-email new-admin@example.com \
  --admin-password 'MySecurePassw0rd!' \
  --force-password
```

Flags override the defaults above and can be combined to rename the root organization or update profile details.

Key configuration options:
- `APP_PORT`: HTTP server port (default: 8080)
- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret key for JWT signing
- `TOKEN_EXPIRATION`: Access token expiration (default: 15m)
- `REFRESH_EXPIRATION`: Refresh token expiration (default: 7d)
- `ENABLE_SWAGGER`: Toggle Swagger UI exposure (enabled by default)
- `SWAGGER_FILE`: Optional path to a custom OpenAPI document (leave blank to use the auto-generated spec)
- `SWAGGER_UI_PATH`: Route prefix for the UI (default: `/swagger/`)
- `SWAGGER_PROTECTED_PREFIXES`: Comma-separated prefixes that should be marked as requiring authentication in the docs (default: `/api`)

### Swagger UI

Once the service is running locally, the interactive documentation is served at:

```
http://localhost:${APP_PORT:-8080}/swagger/
```

The underlying OpenAPI document is generated at runtime from registered routes (`/swagger/openapi.yaml`). Provide `SWAGGER_FILE` to replace it with a hand-crafted spec if needed.

> **Tip:** use the "Authorize" button in the Swagger UI and enter `Bearer <your-access-token>` so protected endpoints include the JWT in the `Authorization` header.

## Core Package Features Applied

This service leverages the following features from the core package:

1. **Middleware Stack**:
   - Request ID generation and propagation
   - Structured logging with Zap
   - Panic recovery
   - Prometheus metrics collection
   - CORS handling
   - Rate limiting (with Redis)

2. **Error Handling**:
   - Standardized error responses
   - HTTP status code mapping
   - Request correlation

3. **Database**:
   - PostgreSQL with GORM ORM
   - Connection pooling
   - Auto-migration
   - Health checks

4. **Configuration**:
   - Environment-based configuration
   - Type-safe settings
   - Validation

5. **Observability**:
   - Health check endpoints
   - Metrics endpoint
   - Structured logging

## Running the Service

### Prerequisites

1. Go 1.22+
2. PostgreSQL database
3. Redis (optional, for rate limiting)

### Development

1. Install dependencies:
```bash
go mod download
```

2. Set up database:
```bash
# Create database
createdb authentication_db

# Database will auto-migrate on startup
```

3. Run the service:
```bash
go run cmd/main.go
```

### Testing with cURL

1. Register a new user:
```bash
curl -X POST http://localhost:8081/api/v1/authentication/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "TestPass123!",
    "first_name": "Test",
    "last_name": "User"
  }'
```

2. Login:
```bash
curl -X POST http://localhost:8081/api/v1/authentication/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "TestPass123!"
  }'
```

3. Refresh token:
```bash
curl -X POST http://localhost:8081/api/v1/authentication/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

4. Check health:
```bash
curl http://localhost:8081/api/v1/authentication/health
curl http://localhost:8081/health/detailed
```

## Docker

Build and run with Docker:

```bash
# Build image
docker build -t lee-tech/authentication-service .

# Run container
docker run -p 8081:8081 \
  -e DATABASE_URL="postgres://user:pass@host:5432/authentication_db" \
  -e JWT_SECRET="your-secret" \
  lee-tech/authentication-service
```

## Security Considerations

1. **Password Security**:
   - Passwords hashed with bcrypt (configurable cost factor)
   - Minimum length enforcement
   - Never exposed in responses

2. **Account Protection**:
   - Login attempt tracking
   - Account lockout after failed attempts
   - Configurable lockout duration

3. **Token Security**:
   - Short-lived access tokens (15 minutes default)
   - Longer refresh tokens (7 days default)
   - Tokens signed with HMAC-SHA256

4. **API Security**:
   - Rate limiting (when Redis enabled)
   - CORS configuration
   - Request ID tracking

## Future Enhancements

- [ ] Email verification
- [ ] Password reset functionality
- [ ] Multi-factor authentication (MFA/2FA)
- [ ] OAuth2/OIDC support (Google, GitHub, etc.)
- [ ] Session management
- [ ] Audit logging
- [ ] Role-based access control integration
# authentication
