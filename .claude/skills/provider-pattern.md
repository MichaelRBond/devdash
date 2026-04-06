# Provider Pattern

Every data source follows this interface pattern:

```go
// types/providers.go
type Provider[T any] interface {
    Fetch(ctx context.Context) ([]T, error)
}
```

Concrete implementations live in `providers/`. They:

1. Accept config in their constructor
2. Handle auth (read token from env)
3. Make API calls
4. Map responses to domain types in `types/`
5. Return clean data or wrapped errors

Example:

```go
// providers/linear.go
type LinearProvider struct {
    apiKey   string
    teamKeys []string
    client   *http.Client
}

func NewLinearProvider(cfg config.LinearConfig) (*LinearProvider, error) {
    key := os.Getenv(cfg.TokenEnv)
    if key == "" {
        return nil, fmt.Errorf("env var %s not set", cfg.TokenEnv)
    }
    return &LinearProvider{
        apiKey:   key,
        teamKeys: cfg.TeamKeys,
        client:   &http.Client{Timeout: 10 * time.Second},
    }, nil
}

func (p *LinearProvider) Fetch(ctx context.Context) ([]types.Task, error) {
    // GraphQL query, parse response, map to []types.Task
}
```

Providers should never import anything from `tui/`. Data flows one way:
provider -> types -> tui panel.
