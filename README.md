# runeplan
OSRS Goal Tracker

## Make Commands

| Command                                | Description                                  |
|----------------------------------------|----------------------------------------------|
| `make`                                 | Generate templates and build (default)       |
| `make generate`                        | Run `templ generate` (required before build) |
| `make build`                           | Compile the server binary                    |
| `make run`                             | Run the server (requires `DATABASE_URL`)     |
| `make test`                            | Run all tests                                |
| `make test-verbose`                    | Run all tests with verbose output            |
| `make test-pkg PKG=./domain/skill/...` | Run tests for a specific package             |
| `make fmt`                             | Format all Go source files                   |
| `make lint`                            | Run `golangci-lint`                          |
| `make vet`                             | Run `go vet`                                 |
| `make deps`                            | Download and verify dependencies             |
| `make tidy`                            | Tidy `go.mod` / `go.sum`                     |
| `make tools`                           | Install dev tools (`templ`, `golangci-lint`) |
| `make clean`                           | Remove generated files and build artifacts   |
| `make help`                            | Show all available commands                  |