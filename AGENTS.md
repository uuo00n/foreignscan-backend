# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/main.go`: application entrypoint and route wiring.
- `internal/`: core app code by layer:
  - `handlers/` HTTP handlers and request/response logic
  - `services/` domain workflows (e.g., detection jobs)
  - `models/` GORM models
  - `database/` PostgreSQL connection and migrations
  - `config/`, `middleware/`, `utils/` shared runtime concerns
- `pkg/utils/`: reusable utility code outside `internal`.
- `docs/`: generated Swagger artifacts (`docs.go`, `swagger.json`, `swagger.yaml`).
- `scripts/`: Docker start/stop scripts and sample data under `scripts/data/`.
- Runtime folders: `uploads/`, `public/` (do not treat as source of truth for code changes).

## Build, Test, and Development Commands
- `go mod download`: install Go dependencies.
- `go run ./cmd/server`: run backend locally (non-Docker).
- `go build -o bin/server ./cmd/server`: build executable.
- `go test ./...`: run all unit tests.
- `./scripts/dev-up.sh` / `./scripts/dev-down.sh`: start/stop dev Docker stack in order (`postgres` then `api`).
- `./scripts/prod-up.sh` / `./scripts/prod-down.sh`: start/stop production-like stack.

## Coding Style & Naming Conventions
- Use standard Go formatting (`gofmt`) and idiomatic Go style.
- Keep package names lowercase; avoid abbreviations unless standard (`cfg`, `ctx` are acceptable).
- Use `CamelCase` for exported symbols, `camelCase` for unexported symbols.
- Follow existing file naming: `snake_case.go` for multi-word files (e.g., `style_images.go`).
- Keep handlers thin; move business logic into `internal/services`.

## Testing Guidelines
- Use Go’s `testing` package with files named `*_test.go` alongside code.
- Prefer table-driven tests for handlers/middleware and edge cases.
- For API changes, add/adjust tests in `internal/handlers/*_test.go`.
- Before opening a PR, run at least `go test ./...` and include the command output summary.

## Commit & Pull Request Guidelines
- Follow Conventional Commit style seen in history: `feat(scope): ...`, `fix(scope): ...`, `refactor: ...`, `docs: ...`, `test: ...`, `chore: ...`.
- Keep commits focused and atomic; avoid mixing refactor + feature + data updates in one commit.
- PRs should include:
  - purpose and impact
  - key files changed
  - validation steps (commands run)
  - API behavior changes (sample request/response when relevant)

## Security & Configuration Tips
- Do not commit secrets or local runtime files (`.env`, `.env.docker`, binaries, temp uploads).
- Use `.env.example` / `.env.docker.example` as templates.
- `FS_DETECT_URL` points to an external detection service; verify reachability when debugging detection endpoints.
