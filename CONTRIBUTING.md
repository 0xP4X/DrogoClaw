# Contributing to DrogonClaw

Thanks for your interest in improving DrogonClaw. This document explains how to
get a change from your machine into `main`.

## Scope

DrogonClaw is an authorized-security-testing platform. Contributions must:

- Stay within the project's scope of defensive and authorized offensive tooling.
- Never add code that performs attacks against systems without explicit scope.
- Respect the existing safety controls (Human-in-the-Loop, sandbox execution,
  OPSEC registry). Do not weaken or bypass them.

For larger features or behavior changes, open an issue first so the scope is
agreed before you start implementing.

## Development Setup

Requirements:

- Go `1.26+`
- Docker (daemon running, for sandbox execution)
- `golangci-lint` for linting

```bash
git clone https://github.com/0xP4X/drogonclaw.git
cd drogonclaw
go mod tidy
make build
```

## Workflow

1. Fork the repository and create a feature branch from `main`.
2. Make your change. Keep commits focused and write clear messages.
3. Run the local checks before pushing:

   ```bash
   make build    # compile the binary
   make lint     # golangci-lint
   make test     # go test -race ./internal/...
   ```

4. Open a pull request against `main` with a clear description of the change
   and the motivation behind it.

## Code Style

- Follow standard Go formatting (`make format` runs `gofmt` and `goimports`).
- Run `go vet` (included in `make vet`) and address reported issues.
- Add or update tests for new behavior, especially in `internal/`.
- Keep documentation in sync (update `README.md` and `docs/` when behavior
  changes).

## Reporting Security Issues

Do not open a public issue for vulnerabilities. Follow the process in
[SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions are licensed under the
GNU AGPL v3, the same license as the project (see [LICENSE](LICENSE)).
