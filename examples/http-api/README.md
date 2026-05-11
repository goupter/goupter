# Goupter HTTP API Example

A complete example demonstrating the Goupter framework features including:

- HTTP Server with Gin
- JWT Authentication
- MySQL Database with GORM
- Memory Cache (can switch to Redis)
- Middleware (Logger, Recovery, CORS, Security, Auth)
- Health Check endpoints
- Metrics and PProf endpoints

## Quick Start

### 1. Start Infrastructure

```bash
docker-compose up -d
```

This starts:
- MySQL 8.0 on port 3306
- Redis 7 on port 6379

### 2. Run the Application

```bash
cd examples/http-api
go run ./cmd/api
```

The server will start on `http://localhost:8080`

## API Endpoints

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Service info |
| GET | `/health` | Liveness check |
| GET | `/ready` | Readiness check |
| GET | `/api/v1/health` | API health check |
| POST | `/api/v1/auth/login` | User login |

### Protected Endpoints (Require JWT)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/me` | Current user info |
| POST | `/api/v1/auth/logout` | Logout |
| GET | `/api/v1/articles` | List articles |
| GET | `/api/v1/articles/:id` | Get article |
| POST | `/api/v1/articles` | Create article |
| PUT | `/api/v1/articles/:id` | Update article |
| DELETE | `/api/v1/articles/:id` | Delete article |

### Admin Endpoints (Require admin role)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/stats` | Admin statistics |

## Usage Examples

### Login

```bash
# Login as admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password123"}'

# Response:
# {"code":0,"data":{"token":"eyJ...","username":"admin","role":"admin"}}
```

### Access Protected Endpoint

```bash
# Get current user info
curl http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer <token>"

# List articles
curl http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer <token>"

# Create article
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Article","content":"Article content"}'
```

### Admin Endpoint

```bash
# Access admin stats (requires admin role)
curl http://localhost:8080/api/v1/admin/stats \
  -H "Authorization: Bearer <admin-token>"
```

## Demo Users

| Username | Password | Role |
|----------|----------|------|
| admin | password123 | admin |
| user1 | password123 | user |

## Configuration

Configuration file: `cmd/api/config/config.yaml`

Key configurations:
- `server.http.port`: HTTP server port (default: 8080)
- `database.*`: MySQL connection settings
- `cache.*`: Cache settings (memory/redis)
- `auth.config.secret_key`: JWT secret key

## Project Structure

```
examples/http-api/
├── cmd/api/
│   ├── config/
│   │   └── config.yaml      # Configuration
│   ├── handler/
│   │   ├── auth.go          # Auth handlers
│   │   ├── article.go       # Article handlers
│   │   └── health.go        # Health handler
│   ├── main.go              # Entry point
│   └── routes.go            # Route registration
├── model/
│   ├── user.go              # User model
│   └── article.go           # Article model
├── util/
│   └── copy.go              # Utility functions
├── docker-compose.yaml      # Docker services
├── init.sql                 # Database schema
├── go.mod
└── README.md
```
