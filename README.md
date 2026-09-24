# adder

[![Release](https://img.shields.io/github/v/tag/lwlee2608/adder?label=release&sort=semver)](https://github.com/lwlee2608/adder/tags)
[![CI](https://github.com/lwlee2608/adder/actions/workflows/ci.yml/badge.svg)](https://github.com/lwlee2608/adder/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lwlee2608/adder.svg)](https://pkg.go.dev/github.com/lwlee2608/adder)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwlee2608/adder)](https://goreportcard.com/report/github.com/lwlee2608/adder)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

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
- Indexed env overrides for slice elements, including elements not in the config file
- Explicit env var binding via `BindEnv()`
- `mapstructure` struct tags for custom field mapping
- Pretty JSON output with sensitive field masking via `PrettyJSON()`
- Singleton and instance-based usage

## Slices from Environment Variables

Slice elements are addressed by index, so `AutomaticEnv()` and `BindEnv()` reach
into lists. Indices beyond what the config file contains are appended, which means
a list can be supplied entirely by the environment.

```yaml
# application.yaml
clients:
  - name: foo
    token: t1
  - name: bar
    token: t2
```

```go
type Config struct {
    Clients []ClientConfig
    Paths   []string
}

type ClientConfig struct {
    Name  string
    Token string
    Auth  AuthConfig
}

type AuthConfig struct {
    Id string
}
```

```bash
CLIENTS_1_TOKEN=t9      # overrides clients[1].token
CLIENTS_2_NAME=baz      # appends clients[2]
CLIENTS_2_AUTH_ID=id-2  # sets clients[2].auth.id
PATHS_0=/one            # builds paths, absent from application.yaml
PATHS_1='/two/*'
```

Notes:

- Appending stops at the first missing index: with `CLIENTS_0_NAME` and `CLIENTS_2_NAME`
  set, only `clients[0]` is added.
- Only a variable that maps to a real field appends an element, so a misspelled
  `CLIENTS_2_TOKNE` is ignored rather than appending a blank client.
- An unindexed env var for a slice key (`CLIENTS=...`) is ignored; only indexed keys apply.
- An unindexed env var for a field *inside* a slice element (`CLIENTS_TOKEN=...`) applies
  to every existing element; an indexed key wins over it for that element. Unindexed keys
  never append elements.
- See [`example/env-slice`](example/env-slice/).

## Mask Sensitive Fields

`PrettyJSON` returns indented JSON while masking string fields tagged with `mask`.

Supported tags:

- `mask:"true"` full mask
- `mask:"first=N"` keep first `N` chars
- `mask:"last=N"` keep last `N` chars
- `mask:"first=N,last=M"` keep both ends
- `mask:"...,preserve=true"` (optional) keeps original length; default masked segment is always 5 `*`

Notes:

- Without `preserve=true`, masked segments are always replaced with exactly 5 `*`.
- If `first+last` overlaps the input and `preserve=true` is not set, masking falls back to `*****`.

```go
type AuthConfig struct {
    Password string `mask:"true"`
    Token    string `mask:"last=3"`
    APIKey   string `mask:"first=2,last=2"`
}

cfg := AuthConfig{Password: "s3cret", Token: "abcdef", APIKey: "ABCDEFGHIJ"}

jsonStr, err := adder.PrettyJSON(cfg)
if err != nil {
    panic(err)
}
fmt.Println(jsonStr)
```

## Examples

See runnable examples in [`example/`](example/).

## License

[MIT](LICENSE)
