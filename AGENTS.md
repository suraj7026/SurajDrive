# Project Architecture & Conventions

- **Repo scope**: This repo is the Go backend only. The Next.js frontend lives in the [`suraj-drive-webui`](https://github.com/suraj7026/suraj-drive-webui) repo.
- **Module path**: `surajdrive/backend` (see `backend/go.mod`). All internal packages import via `surajdrive/backend/internal/...`.
- **Layout**: Standard Go layout — `backend/cmd/server/` for the entrypoint, `backend/internal/` for everything else (`auth`, `config`, `handler`, `middleware`, `model`, `storage`).
- **Router**: [`go-chi/chi`](https://github.com/go-chi/chi) is the only HTTP router. Wire all new routes in `backend/cmd/server/main.go`.
- **Auth**: JWT in an HTTP-only cookie. Use `appmiddleware.RequireAuth(cfg.JWT.Secret)` to gate routes.
- **Storage**: MinIO via the `internal/storage` client. Per-user bucket naming uses `minio.bucket_prefix` plus the user identifier — do not hardcode bucket names.
- **Config**: `backend/config.yaml` provides defaults; env vars (`SECTION_KEY`, e.g. `JWT_SECRET`, `MINIO_ENDPOINT`) override them. Never commit real secrets.
- **CORS**: Driven by `server.frontend_url` (default `http://localhost:4000`). Update this when deploying the frontend elsewhere.
- **Ports**: Backend listens on `4001` by default; frontend (separate repo) runs on `4000`.
- **Logs**: Use `github.com/rs/zerolog/log` for structured logging — no `fmt.Println` in handlers.
