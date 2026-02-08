package main

import (
	"encoding/json"
	"fmt"

	"github.com/lwlee2608/adder"
)

type Config struct {
	Log  LogConfig
	Http HttpConfig
	Db   DbConfig
}

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

func main() {
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

	configJSON, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Config loaded:")
	fmt.Println(string(configJSON))
}
