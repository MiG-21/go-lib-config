# go-lib-config

## Usage

### ENV reader

```go
import (
    "time"

    libConfig "github.com/MiG-21/go-lib-config"
)

func main() {
    var cfg Config
    // refresh config per 1 minute
    service := libConfig.NewConfigService(1 * time.Minute)
    reader := libConfig.NewEnvReader()
    if err := service.Start(&cfg, nil, reader); err != nil {
        // some error handler
    }
    defer service.Stop()
}
```

### Vault reader by token

```go
import (
    "time"

    libConfig "github.com/MiG-21/go-lib-config"
)

func defaultPathFormatter(secret string) string {
    parts := []string{"secret", "data"}
    parts = append(parts, strings.Trim(secret, "/"))
    return strings.Join(parts, "/")
}

func main() {
    var cfg Config
    service := libConfig.NewConfigService(1 * time.Minute)
    auth := libConfig.NewVaultTokenAuth("token")
    vault := libConfig.NewStorageVault(auth, "vault address", "data")
    reader := libConfig.NewVaultReaderWithFormatter(vault, defaultPathFormatter)
    if err := service.Start(&cfg, nil, reader); err != nil {
        // some error handler
    }
    defer service.Stop()
}
```

### Vault reader by K8s

```go
import (
    "time"

    libConfig "github.com/MiG-21/go-lib-config"
)

func defaultPathFormatter(secret string) string {
    parts := []string{"secret", "data"}
    parts = append(parts, strings.Trim(secret, "/"))
    return strings.Join(parts, "/")
}

func main() {
    var cfg Config
    service := libConfig.NewConfigService(1 * time.Minute)
    auth := libConfig.NewVaultK8sAuth("vault address", "auth endpoint", "token path", "role")
    vault := libConfig.NewStorageVault(auth, "vault address", "data")
    reader := libConfig.NewVaultReaderWithFormatter(vault, defaultPathFormatter)
    if err := service.Start(&cfg, nil, reader); err != nil {
        // some error handler
    }
    defer service.Stop()
}
```

### Assigning validator

```go
import (
    "github.com/go-playground/validator/v10"

    libConfig "github.com/MiG-21/go-lib-config"
)

// Validator wrapper example
type Validator struct {
    validator *validator.Validate
}

func (cv *Validator) Validate(i interface{}) error {
    return cv.validator.Struct(i)
}

func main() {
    var cfg Config
    service := libConfig.NewConfigService(1 * time.Minute)
    // assign validator
    service.Validator = &Validator{validator: validator.New()}
    // ....
}
```
