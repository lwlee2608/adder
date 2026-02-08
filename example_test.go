package adder_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/adder"
)

func writeConfig(dir, content string) {
	path := filepath.Join(dir, "application.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
}

func Example() {
	dir, _ := os.MkdirTemp("", "adder")
	defer os.RemoveAll(dir)
	writeConfig(dir, `
server:
  host: localhost
  port: 8080
`)

	type Config struct {
		Server struct {
			Host string
			Port uint
		}
	}

	a := adder.New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)

	if err := a.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg Config
	if err := a.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	fmt.Println(cfg.Server.Host)
	fmt.Println(cfg.Server.Port)
	// Output:
	// localhost
	// 8080
}

func ExampleAdder_AutomaticEnv() {
	dir, _ := os.MkdirTemp("", "adder")
	defer os.RemoveAll(dir)
	writeConfig(dir, `
http:
  port: 8080
`)

	os.Setenv("HTTP_PORT", "9090")
	defer os.Unsetenv("HTTP_PORT")

	type Config struct {
		Http struct {
			Port uint
		}
	}

	a := adder.New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	if err := a.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg Config
	if err := a.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	fmt.Println(cfg.Http.Port)
	// Output:
	// 9090
}

func ExampleAdder_BindEnv() {
	dir, _ := os.MkdirTemp("", "adder")
	defer os.RemoveAll(dir)
	writeConfig(dir, `
db:
  url: postgres://localhost/mydb
`)

	os.Setenv("DATABASE_URL", "postgres://prod/mydb")
	defer os.Unsetenv("DATABASE_URL")

	type Config struct {
		Db struct {
			Url string
		}
	}

	a := adder.New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)
	a.BindEnv("db.url", "DATABASE_URL")

	if err := a.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg Config
	if err := a.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	fmt.Println(cfg.Db.Url)
	// Output:
	// postgres://prod/mydb
}

func ExampleAdder_Unmarshal_slices() {
	dir, _ := os.MkdirTemp("", "adder")
	defer os.RemoveAll(dir)
	writeConfig(dir, `
app:
  allowed_origins:
    - https://app.example.com
    - https://admin.example.com
`)

	type Config struct {
		App struct {
			AllowedOrigins []string `mapstructure:"allowed_origins"`
		}
	}

	a := adder.New()
	a.SetConfigName("application")
	a.SetConfigType("yaml")
	a.AddConfigPath(dir)

	if err := a.ReadInConfig(); err != nil {
		panic(err)
	}

	var cfg Config
	if err := a.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	for _, origin := range cfg.App.AllowedOrigins {
		fmt.Println(origin)
	}
	// Output:
	// https://app.example.com
	// https://admin.example.com
}
