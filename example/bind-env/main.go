package main

import (
	"encoding/json"
	"fmt"

	"github.com/lwlee2608/adder"
)

type Config struct {
	Db DbConfig
}

type DbConfig struct {
	Url    string
	Schema string
}

func main() {
	adder.SetConfigName("application")
	adder.AddConfigPath(".")
	adder.SetConfigType("yaml")

	// Explicitly bind config keys to specific environment variables.
	// This is useful when env var names don't follow a simple pattern.
	adder.BindEnv("db.url", "DATABASE_URL")
	adder.BindEnv("db.schema", "DATABASE_SCHEMA")

	if err := adder.ReadInConfig(); err != nil {
		panic(err)
	}

	var config Config
	if err := adder.Unmarshal(&config); err != nil {
		panic(err)
	}

	configJSON, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Config loaded:")
	fmt.Println(string(configJSON))
}
