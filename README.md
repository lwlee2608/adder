# adder

A lightweight configuration library for Go, inspired by [viper](https://github.com/spf13/viper).

Adder reads YAML config files and unmarshals them into Go structs, with support for environment variable overrides.

## Installation

```bash
go get github.com/lwlee2608/adder
```

## Quick Start

Define a config struct and a YAML file:

```yaml
# application.yaml
server:
  host: localhost
  port: 8080
```

```go
type Config struct {
    Server struct {
        Host string
        Port uint
    }
}

adder.SetConfigName("application")
adder.AddConfigPath(".")
adder.SetConfigType("yaml")

if err := adder.ReadInConfig(); err != nil {
    panic(err)
}

var config Config
if err := adder.Unmarshal(&config); err != nil {
    panic(err)
}
```

## Features

### YAML Configuration

Load configuration from YAML files. Multiple search paths are supported:

```go
adder.AddConfigPath(".")
adder.AddConfigPath("/etc/myapp")
```

### Environment Variable Overrides

Override any config value with environment variables. Enable automatic mapping with `AutomaticEnv()` and a key replacer:

```go
adder.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
adder.AutomaticEnv()
```

This maps config keys to env vars by converting dots to underscores and uppercasing:

| Config Key   | Environment Variable |
|--------------|----------------------|
| `http.port`  | `HTTP_PORT`          |
| `db.url`     | `DB_URL`             |
| `log.level`  | `LOG_LEVEL`          |

### Explicit Environment Binding

Bind specific config keys to environment variables when the naming convention doesn't match:

```go
adder.BindEnv("db.url", "DATABASE_URL")
```

### Struct Tag Support

Use `mapstructure` tags for custom field name mapping:

```go
type AppConfig struct {
    AllowedOrigins []string `mapstructure:"allowed_origins"`
}
```

### Supported Types

- `string`, `bool`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `[]string`, `[]int`, `[]int64`
- Nested structs

### Multiple Instances

Use the default singleton or create isolated instances:

```go
// Singleton (package-level functions)
adder.SetConfigName("application")

// Instance-based
a := adder.New()
a.SetConfigName("application")
```

## API

| Function | Description |
|----------|-------------|
| `New()` | Create a new Adder instance |
| `SetConfigName(name)` | Set config filename (without extension) |
| `SetConfigType(typ)` | Set config file type (`"yaml"` or `"yml"`) |
| `AddConfigPath(path)` | Add a directory to search for config files |
| `SetEnvKeyReplacer(r)` | Set a `strings.Replacer` for env var name mapping |
| `AutomaticEnv()` | Enable automatic env var overrides |
| `BindEnv(key, envVar)` | Bind a config key to a specific env var |
| `ReadInConfig()` | Load config from the file system |
| `Unmarshal(v)` | Unmarshal config into a struct |

All functions are available both as package-level functions (using a default instance) and as methods on `*Adder`.

## Examples

See the [example/](example/) directory:

- **[basic](example/basic/)** — Load config from a YAML file
- **[env-override](example/env-override/)** — Override config values with environment variables
- **[bind-env](example/bind-env/)** — Bind config keys to specific env var names

Run any example:

```bash
cd example/basic
go run main.go

# With env overrides
cd example/env-override
HTTP_PORT=9090 LOG_LEVEL=debug go run main.go

# With explicit bindings
cd example/bind-env
DATABASE_URL=postgres://prod:5432/db go run main.go
```

## License

[MIT](LICENSE)
