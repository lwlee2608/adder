package main

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/lwlee2608/adder"
)

type Config struct {
	Log LogConfig
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

func main() {
	_ = godotenv.Load()

	adder.SetConfigName("application")
	adder.AddConfigPath(".")
	adder.SetConfigType("yaml")
	adder.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	adder.AutomaticEnv()
}
