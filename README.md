# go-lib-config

## Usage

### Service initiation

if duration == 0 config refresh loop will not been started

duration validation is not provided, so this is entirely your responsibility, keep in mind that too small an interval can lead to unforeseen consequences

```go
// turn on logging, by default turned off
libConfig.Verbose = true
// refresh interval
duration := 1 * time.Minute
// get service instance
service := libConfig.NewConfigService(duration)
// start service
valid, err := service.Start(&cfg, cb, reader)
```

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
    // this callback will be called on every config update
    cb = func(err error) {
        // some error handler
    }
    if valid, err := service.Start(&cfg, cb, reader); err != nil {
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
    if valid, err := service.Start(&cfg, nil, reader); err != nil {
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
    if valid, err := service.Start(&cfg, nil, reader); err != nil {
        // some error handler
    }
    defer service.Stop()
}
```

### Assigning validator

Validator should implement interface

```go
type Validator interface {
    Validate(i interface{}) error
}
```

#### Example

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

### More than one reader

the priority of the readers is related to the order, each next is higher than the previous one, the last one has the highest priority

```go
func main() {
    var cfg Config
    service := libConfig.NewConfigService(1 * time.Minute)
    reader1 := Reader1{}
    reader2 := Reader2{}
    if valid, err := service.Start(&cfg, nil, reader1, reader2); err != nil {
        // some error handler
    }
}
```

### Custom reader

Custom reader can be implemented in accordance with interface `Reader`

```go
type Reader interface {
    Read(cfg interface{}) error
}
```

#### Example

```go
type CustomReader struct {

}

func (r *CustomReader) Read(cfg interface{}) error {
    // some implementation
}

func main() {
    var cfg Config
    service := libConfig.NewConfigService(1 * time.Minute)
    reader := CustomReader{}
    if valid, err := service.Start(&cfg, nil, reader); err != nil {
        // some error handler
    }
    defer service.Stop()
}
```

### Custom logger

```go
LibLogger = func(i ...interface{}) {
	// custom implementation
}
```
