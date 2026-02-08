package adder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
