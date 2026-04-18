package adder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Log  testLogConfig
	Http testHTTPConfig
	Db   testDBConfig
}

type testLogConfig struct {
	Level string
}

type testHTTPConfig struct {
	Port uint
}

type testDBConfig struct {
	URL    string `mapstructure:"url"`
	Schema string
}

func TestUnmarshalUintFromYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	content := `http:
  port: 8080
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)

	require.NoError(t, a.ReadInConfig())

	var cfg testConfig
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, uint(8080), cfg.Http.Port)
}

func TestReadInConfig_WithYamlTypeFindsYmlFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yml")
	content := `http:
  port: 8080
`

	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("Yaml")
	a.AddConfigPath(dir)

	require.NoError(t, a.ReadInConfig())

	var cfg testConfig
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, uint(8080), cfg.Http.Port)
}

func TestAutomaticEnvOverrideUint(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	content := `http:
  port: 8080
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("HTTP_PORT", "9091")

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	require.NoError(t, a.ReadInConfig())

	var cfg testConfig
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, uint(9091), cfg.Http.Port)
}

func TestBindEnvOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	content := `db:
  url: postgres://from-config
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("DATABASE_URL", "postgres://from-env")

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)

	require.NoError(t, a.BindEnv("db.url", "DATABASE_URL"))

	require.NoError(t, a.ReadInConfig())

	var cfg testConfig
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, "postgres://from-env", cfg.Db.URL)
}

func TestBindEnvOverride_MissingSectionInYAML(t *testing.T) {
	type apiConfig struct {
		ApiKey string
	}
	type config struct {
		Api apiConfig
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	// No "api:" section in the YAML at all
	content := `log:
  level: info
`

	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))
	t.Setenv("MY_API_KEY", "secret-key-from-env")

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)

	require.NoError(t, a.BindEnv("api.apikey", "MY_API_KEY"))
	require.NoError(t, a.ReadInConfig())

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, "secret-key-from-env", cfg.Api.ApiKey)
}

func TestReadInConfigErrors(t *testing.T) {
	t.Run("missing config name", func(t *testing.T) {
		a := New()
		a.SetConfigType("yaml")
		a.AddConfigPath(t.TempDir())

		err := a.ReadInConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config name not set")
	})

	t.Run("missing config file", func(t *testing.T) {
		a := New()
		a.SetConfigName("application")
		a.SetConfigType("yaml")
		a.AddConfigPath(t.TempDir())

		err := a.ReadInConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config file not found")
	})

	t.Run("unsupported config type", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "application.toml")
		if err := os.WriteFile(configPath, []byte(`key = "value"
`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		a := New()
		a.SetConfigName("application")
		a.SetConfigType("toml")
		a.AddConfigPath(dir)

		err := a.ReadInConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported config type")
	})
}

func TestAutomaticEnvOverrideUint_InvalidValue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	content := `http:
  port: 8080
`

	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))
	t.Setenv("HTTP_PORT", "not-a-number")

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	require.NoError(t, a.ReadInConfig())

	var cfg testConfig
	err := a.Unmarshal(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid syntax")
}

func TestAutomaticEnvOverrideFromYAMLDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	content := `log:
  level: info
http:
  port: 8080
db:
  url: postgres://from-config
  schema: public
`

	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("HTTP_PORT", "9091")
	t.Setenv("DB_SCHEMA", "tenant_alpha")

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	require.NoError(t, a.ReadInConfig())

	var cfg testConfig
	require.NoError(t, a.Unmarshal(&cfg))

	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, uint(9091), cfg.Http.Port)
	assert.Equal(t, "postgres://from-config", cfg.Db.URL)
	assert.Equal(t, "tenant_alpha", cfg.Db.Schema)
}

func TestCaseInsensitiveYAMLKeys(t *testing.T) {
	type config struct {
		BaseURL string
	}

	tests := []struct {
		name    string
		yamlKey string
	}{
		{"lowercase", "baseurl"},
		{"camelCase", "baseUrl"},
		{"original", "baseURL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			content := tc.yamlKey + ": https://example.com\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))

			a := New()
			a.SetConfigName("application")
			a.SetConfigType("yaml")
			a.AddConfigPath(dir)
			require.NoError(t, a.ReadInConfig())

			var cfg config
			require.NoError(t, a.Unmarshal(&cfg))
			assert.Equal(t, "https://example.com", cfg.BaseURL)
		})
	}
}

func TestEnvVarExpansionInYAML(t *testing.T) {
	type something struct {
		ApiKey string
	}
	type config struct {
		Something something
	}

	t.Run("expands ${VAR} syntax", func(t *testing.T) {
		dir := t.TempDir()
		content := `something:
  apikey: ${SOMETHING_API_KEY}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))
		t.Setenv("SOMETHING_API_KEY", "my-secret-key")

		a := New()
		a.SetConfigName("application")
		a.SetConfigType("yaml")
		a.AddConfigPath(dir)

		require.NoError(t, a.ReadInConfig())

		var cfg config
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "my-secret-key", cfg.Something.ApiKey)
	})

	t.Run("unset var expands to empty string", func(t *testing.T) {
		dir := t.TempDir()
		content := `something:
  apikey: ${UNSET_VAR_12345}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))

		a := New()
		a.SetConfigName("application")
		a.SetConfigType("yaml")
		a.AddConfigPath(dir)

		require.NoError(t, a.ReadInConfig())

		var cfg config
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "", cfg.Something.ApiKey)
	})

	t.Run("mixed literal and env var", func(t *testing.T) {
		type db struct {
			URL string `mapstructure:"url"`
		}
		type dbConfig struct {
			Db db
		}

		dir := t.TempDir()
		content := `db:
  url: postgres://${DB_HOST}:5432/mydb
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))
		t.Setenv("DB_HOST", "prod-server")

		a := New()
		a.SetConfigName("application")
		a.SetConfigType("yaml")
		a.AddConfigPath(dir)

		require.NoError(t, a.ReadInConfig())

		var cfg dbConfig
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "postgres://prod-server:5432/mydb", cfg.Db.URL)
	})

	t.Run("bare $VAR is not expanded", func(t *testing.T) {
		dir := t.TempDir()
		content := `something:
  apikey: $SOMETHING_API_KEY
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))
		t.Setenv("SOMETHING_API_KEY", "my-secret-key")

		a := New()
		a.SetConfigName("application")
		a.SetConfigType("yaml")
		a.AddConfigPath(dir)

		require.NoError(t, a.ReadInConfig())

		var cfg config
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "$SOMETHING_API_KEY", cfg.Something.ApiKey)
	})

	t.Run("literal dollar signs are preserved", func(t *testing.T) {
		dir := t.TempDir()
		content := `something:
  apikey: p@$$word
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))

		a := New()
		a.SetConfigName("application")
		a.SetConfigType("yaml")
		a.AddConfigPath(dir)

		require.NoError(t, a.ReadInConfig())

		var cfg config
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "p@$$word", cfg.Something.ApiKey)
	})
}

func TestUnmarshalStructSliceFromYAML(t *testing.T) {
	type item struct {
		Name  string
		Count int
	}
	type config struct {
		Items []item
	}

	dir := t.TempDir()
	content := "items:\n  - Name: foo\n    Count: 10\n  - name: bar\n    count: 20\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	require.NoError(t, a.ReadInConfig())

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))

	require.Len(t, cfg.Items, 2)
	assert.Equal(t, "foo", cfg.Items[0].Name)
	assert.Equal(t, 10, cfg.Items[0].Count)
	assert.Equal(t, "bar", cfg.Items[1].Name)
	assert.Equal(t, 20, cfg.Items[1].Count)
}

func TestUnmarshalStringArrayFromYAML(t *testing.T) {
	type appConfig struct {
		AllowedOrigins []string `mapstructure:"allowed_origins"`
	}

	type config struct {
		App appConfig
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "application.yaml")
	content := `app:
  allowed_origins:
    - https://app.local
    - https://admin.local
`

	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)

	require.NoError(t, a.ReadInConfig())

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, []string{"https://app.local", "https://admin.local"}, cfg.App.AllowedOrigins)
}

func TestUnmarshalMapStringString(t *testing.T) {
	type server struct {
		Name    string
		URL     string            `mapstructure:"url"`
		Headers map[string]string `mapstructure:"headers"`
	}
	type config struct {
		Servers []server
	}

	a := newTestAdder(t, `
servers:
  - name: "api"
    url: "https://api.example.com"
    headers:
      Authorization: "Bearer token"
      X-Custom: "value"
  - name: "plain"
    url: "https://plain.example.com"
`)

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))

	require.Len(t, cfg.Servers, 2)
	assert.Equal(t, "api", cfg.Servers[0].Name)
	assert.Equal(t, map[string]string{
		"Authorization": "Bearer token",
		"X-Custom":      "value",
	}, cfg.Servers[0].Headers)
	assert.Equal(t, "plain", cfg.Servers[1].Name)
	assert.Nil(t, cfg.Servers[1].Headers)
}

func TestUnmarshalMapStringStringTopLevel(t *testing.T) {
	type config struct {
		Labels map[string]string
	}

	a := newTestAdder(t, `
labels:
  env: production
  team: backend
`)

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, map[string]string{
		"env":  "production",
		"team": "backend",
	}, cfg.Labels)
}

func TestUnmarshalMapStringStringNonStringValues(t *testing.T) {
	type config struct {
		Headers map[string]string
	}

	a := newTestAdder(t, `
headers:
  X-Retry-Count: 3
  X-Debug: true
  X-Ratio: 3.14
  X-Name: hello
`)

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, map[string]string{
		"X-Retry-Count": "3",
		"X-Debug":       "true",
		"X-Ratio":       "3.14",
		"X-Name":        "hello",
	}, cfg.Headers)
}

func TestUnmarshalUnsupportedMapType(t *testing.T) {
	type config struct {
		Counts map[string]int64
	}

	a := newTestAdder(t, `
counts:
  a: 1
  b: 2
`)

	var cfg config
	err := a.Unmarshal(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported map type")
}

func TestSetConfigFile(t *testing.T) {
	t.Run("exact yaml path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "custom-config.yaml")
		require.NoError(t, os.WriteFile(path, []byte("log:\n  level: debug\n"), 0o644))

		a := New()
		a.SetConfigFile(path)
		require.NoError(t, a.ReadInConfig())

		var cfg testConfig
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "debug", cfg.Log.Level)
	})

	t.Run("extensionless path defaults to yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config")
		require.NoError(t, os.WriteFile(path, []byte("log:\n  level: warn\n"), 0o644))

		a := New()
		a.SetConfigFile(path)
		require.NoError(t, a.ReadInConfig())

		var cfg testConfig
		require.NoError(t, a.Unmarshal(&cfg))
		assert.Equal(t, "warn", cfg.Log.Level)
	})

	t.Run("missing file returns error", func(t *testing.T) {
		a := New()
		a.SetConfigFile("/nonexistent/config.yaml")
		err := a.ReadInConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config file not found")
	})
}

func TestUnmarshalDurationFromYAML(t *testing.T) {
	type httpConfig struct {
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
		IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	}
	type config struct {
		Http httpConfig
	}

	a := newTestAdder(t, `
http:
  read_timeout: 5s
  write_timeout: 1m30s
  idle_timeout: 2h
`)

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, 5*time.Second, cfg.Http.ReadTimeout)
	assert.Equal(t, 90*time.Second, cfg.Http.WriteTimeout)
	assert.Equal(t, 2*time.Hour, cfg.Http.IdleTimeout)
}

func TestUnmarshalDurationNumericNanoseconds(t *testing.T) {
	type config struct {
		Timeout time.Duration
	}

	a := newTestAdder(t, `
timeout: 1500000000
`)

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, 1500*time.Millisecond, cfg.Timeout)
}

func TestDurationEnvOverride(t *testing.T) {
	type httpConfig struct {
		ReadTimeout time.Duration `mapstructure:"read_timeout"`
	}
	type config struct {
		Http httpConfig
	}

	dir := t.TempDir()
	content := `http:
  read_timeout: 5s
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))
	t.Setenv("HTTP_READ_TIMEOUT", "250ms")

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	require.NoError(t, a.ReadInConfig())

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, 250*time.Millisecond, cfg.Http.ReadTimeout)
}

func TestUnmarshalDurationInvalidString(t *testing.T) {
	type config struct {
		Timeout time.Duration
	}

	a := newTestAdder(t, `
timeout: not-a-duration
`)

	var cfg config
	err := a.Unmarshal(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

func TestUnmarshalDurationSlice(t *testing.T) {
	type config struct {
		Backoffs []time.Duration
	}

	a := newTestAdder(t, `
backoffs:
  - 100ms
  - 500ms
  - 2s
`)

	var cfg config
	require.NoError(t, a.Unmarshal(&cfg))
	assert.Equal(t, []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		2 * time.Second,
	}, cfg.Backoffs)
}

func newTestAdder(t *testing.T, content string) *Adder {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0o644))

	a := New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	require.NoError(t, a.ReadInConfig())
	return a
}
