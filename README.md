# CoolVibes Core

A Go-based backend application with WebSocket support, PostgreSQL database, and JWT authentication.

## Features

- **HTTP action API** with Fiber v3
- **WebSocket support** using Socket.IO
- **PostgreSQL database** with GORM ORM
- **JWT authentication** for secure API access
- **CORS support** for cross-origin requests
- **Static file serving**
- **Environment configuration** with dotenv

## Prerequisites

- Go 1.26.3 or higher
- PostgreSQL 15+ with PostGIS extension
- Git

## Installation

### 1. Clone the repository
```bash
git clone <repository-url>
cd core
```

### 2. Install PostgreSQL and PostGIS

**Ubuntu/Debian:**
```bash
sudo apt-get install postgresql-15-postgis-3
# or for PostgreSQL 17
sudo apt install postgis postgresql-17-postgis-3
```

**macOS:**
```bash
brew install postgresql postgis
```

```bash
brew install golangci-lint
```

### 3. Set up environment variables
```bash
cp env.sample .env
# Edit .env with your database credentials and other settings
# USER_JWT_SECRET is required and must be at least 32 bytes.
# Generate one with: openssl rand -base64 48
```

User JWT validation is fail-closed (HS256, issuer/subject/identity and temporal
claims are required) with 30 seconds of clock-skew tolerance. Tokens issued by
older builds without `iat`/`nbf` claims are intentionally rejected, so users
must sign in again after deploying this change. Rotate `USER_JWT_SECRET` through
your secret manager; never commit it to the repository.

### 4. Install Go dependencies
```bash
go mod download
```

### 5. Initialize the database and start

Run migrations and seed data together on first install:

```bash
./start.sh -install
```

Run migration or seed independently when needed:

```bash
./start.sh -migrate
```

The migration also installs required report-kind reference data and the
database constraints used by atomic location/engagement writes.

```bash
./start.sh -seed
```

Start the server:

```bash
./start.sh
```

The launcher fingerprints the Go source tree, builds an optimized native
binary only when it changes, and executes the cached binary on later starts.

Grant moderation access to an existing, usable account by public ID:

```bash
./start.sh -grant-admin 123456789
./start.sh -grant-moderator 987654321
```

### 6. Lint
```bash
golangci-lint run ./...
```

### 7. Dependency Composition

Dependency wiring is maintained manually in `infrastructure/bootstrap/bootstrap.go`.

### 8. Coverage
```bash
go test -race -cover ./...
```

### 9. Test
```bash
go test ./...
```




## Project Structure
```
core/
├── adapters/inbound/        # Fiber HTTP and MCP input adapters
├── application/
│   ├── ports/               # Repository and external-system boundaries
│   ├── types/               # Persistence-free application DTOs
│   ├── legacyviews/         # Quarantined legacy response projections
│   └── usecases/            # Application orchestration
├── domain/                  # Framework-independent business rules
├── infrastructure/
│   ├── bootstrap/           # Composition root
│   ├── repositories/        # GORM outbound adapters
│   └── ...                  # Auth, media, GeoIP, socket and other adapters
├── models/                  # Legacy GORM/schema compatibility boundary
├── workers/                 # Background processors
├── static/                  # Static files served by the application
├── main.go                  # Process lifecycle and maintenance commands
└── go.mod
```

The enforced dependency direction and remaining migration debt are documented
in `docs/ddd-audit.md`.

## API Endpoints

- `GET /` - Home endpoint
- `POST /packet` - Main packet handler for authentication and other actions
- `GET /static/*` - Static file serving

## Authentication

The application uses JWT tokens for authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

Only HS256 user tokens with the expected issuer/subject and complete time
claims are accepted. Startup fails closed when `USER_JWT_SECRET` is missing or
shorter than 32 bytes.

## WebSocket

WebSocket server runs alongside the HTTP server and handles real-time communication.

## Development

To run the application in development mode:

```bash
./start.sh
```

The server will start on the port specified in your `.env` file.
File uploads have no application-level size limit. Large multipart files are
spooled to temporary files instead of being retained entirely in memory.
Private and chat upload URLs require an authorized bearer token.
If a reverse proxy or ingress is used, its request-body limit must also be
disabled; that limit is outside this process. Configure `TRUSTED_PROXIES` with
an explicit comma-separated proxy IP/CIDR allowlist before relying on forwarded
client IP headers.

For an Nginx API location, disable both the byte cap and request buffering so
the body can flow through to Fiber's disk-backed multipart parser:

```nginx
client_max_body_size 0;
proxy_request_buffering off;
```

Private-photo album/batch item counts remain domain cardinality rules; they do
not inspect or cap the number of bytes in an uploaded file.

## Dependencies

- **Fiberv3** - HTTP router and URL matcher
- **GORM** - ORM library for Go
- **PostgreSQL Driver** - Database driver for PostgreSQL
- **Socket.IO** - WebSocket library
- **JWT** - JSON Web Token implementation
- **CORS** - Cross-Origin Resource Sharing middleware


## License
This project is free to use, open for everyone, and can be developed by anyone.



## Update Homebrew
```bash
brew update
brew install postgresql
brew install postgis
brew services start postgresql
brew services list

brew services start postgresql
psql postgres

ALTER ROLE postgres WITH PASSWORD 'yourownpassword';

brew services restart postgresql
```

## Installation

server {
    listen 80;
    server_name socket.coolvibes.lgbt socket.coolvibes.app socket.coolvibes.io;

    location /socket.io/ {
        proxy_pass http://127.0.0.1:3002;
        proxy_http_version 1.1;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    location /health {
        return 200 "OK";
    }
}

sudo systemctl reload nginx


Test:

go test ./...
