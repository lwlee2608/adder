package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/lwlee2608/adder"
)

type Config struct {
	Log  LogConfig
	Http HttpConfig
	Db   DbConfig
	App  AppConfig
}

const (
	LOG_LEVEL_ERROR   = "ERROR"
	LOG_LEVEL_WARNING = "WARNING"
	LOG_LEVEL_INFO    = "INFO"
	LOG_LEVEL_DEBUG   = "DEBUG"
)

type LogConfig struct {
	Level string
}
type HttpConfig struct {
	Port uint
}
type DbConfig struct {
	Url    string
	Schema string
}

type AppConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

func main() {
	_ = godotenv.Load()

	adder.SetConfigName("application")
	adder.AddConfigPath(".")
	adder.SetConfigType("yaml")
	adder.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	adder.AutomaticEnv()

	if err := adder.ReadInConfig(); err != nil {
		panic(err)
	}

	var config Config

	if err := adder.Unmarshal(&config); err != nil {
		panic(err)
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println("Config loaded:")
	fmt.Println(string(configJSON))
}
