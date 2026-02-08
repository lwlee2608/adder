# adder

A lightweight configuration library for Go, inspired by [viper](https://github.com/spf13/viper).

Adder reads YAML config files and unmarshals them into Go structs, with support for environment variable overrides.

## Installation

```bash
go get github.com/lwlee2608/adder
```

## Quick Start

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

- Case-insensitive YAML key matching
- YAML configuration with multiple search paths
- Automatic environment variable overrides via `AutomaticEnv()`
- Explicit env var binding via `BindEnv()`
- `mapstructure` struct tags for custom field mapping
- Singleton and instance-based usage

## Examples

See the [example/](example/) directory:

- **[basic](example/basic/)** — Load config from a YAML file
- **[env-override](example/env-override/)** — Override config values with environment variables
- **[bind-env](example/bind-env/)** — Bind config keys to specific env var names

## License

[MIT](LICENSE)
