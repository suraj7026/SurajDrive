# SDrive API

Go HTTP API that powers the [SDrive WebUI](https://github.com/suraj7026/suraj-drive-webui). Built on:

- [`chi`](https://github.com/go-chi/chi) router
- Google OAuth2 + JWT session cookies
- MinIO / S3-compatible object storage for file content
- Per-user buckets with presigned upload/download URLs

The frontend lives in a separate repo: [`suraj-drive-webui`](https://github.com/suraj7026/suraj-drive-webui).

## Layout

```
backend/
├── cmd/server/main.go         # HTTP entrypoint, route wiring
├── config.yaml                # Default config (override via env)
├── Dockerfile                 # Multi-stage build
├── docker-compose.yml         # Runs backend image, mounts config.yaml
├── deploy/nginx/              # Reverse proxy configs
├── go.mod / go.sum
└── internal/
    ├── auth/                  # Google OAuth + JWT helpers
    ├── config/                # Config loading
    ├── handler/               # auth, files, folders, upload, download, search
    ├── middleware/            # CORS, auth
    ├── model/                 # Shared API types
    └── storage/               # MinIO client
```

## Local Setup

1. Install Go 1.22+.
2. From the `backend/` directory, copy `config.yaml` and fill in real Google OAuth credentials, JWT secret, and MinIO access keys (or set the equivalent env vars — see below).
3. Run the server:

   ```bash
   cd backend
   go run ./cmd/server
   ```

The server listens on `http://localhost:4001` by default and expects the frontend at `http://localhost:4000`.

## Docker

```bash
cd backend
docker compose up --build
```

This mounts `config.yaml` into the container and exposes port `4001`.

## Configuration

`backend/config.yaml` provides defaults; every value can be overridden by an environment variable using the `SECTION_KEY` upper-case naming convention (e.g. `SERVER_PORT`, `MINIO_ENDPOINT`, `JWT_SECRET`).

Key settings:

| Section  | Notes                                                                 |
| -------- | --------------------------------------------------------------------- |
| `server` | `port` (4001), `frontend_url` (used for CORS + OAuth redirect).       |
| `google` | OAuth client ID/secret, callback URL, optional `allowed_domain`.      |
| `jwt`    | Signing secret, session expiry in hours.                              |
| `minio`  | Endpoint, public endpoint, access/secret keys, bucket prefix, region. |

## API Surface

Public:

- `GET  /api/health`
- `GET  /api/auth/google/login`
- `GET  /api/auth/google/callback`
- `POST /api/auth/logout`

Authenticated (JWT cookie):

- `GET    /api/auth/me`
- `GET    /api/files`
- `POST   /api/files/upload`
- `DELETE /api/files`
- `POST   /api/files/copy`
- `GET    /api/files/presign/download`
- `GET    /api/files/presign/upload`
- `POST   /api/folders`
- `DELETE /api/folders`
- `GET    /api/search`

## Notes

- The backend must allow the frontend origin in CORS via `server.frontend_url` / `SERVER_FRONTEND_URL`.
- Each authenticated user is mapped to a dedicated MinIO bucket (`bucket_prefix` + user identifier).
