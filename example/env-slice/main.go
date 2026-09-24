package main

import (
	"fmt"
	"strings"

	"github.com/lwlee2608/adder"
)

type Config struct {
	Log     LogConfig
	Clients []ClientConfig
	Paths   []string
}

type LogConfig struct {
	Level string
}

type ClientConfig struct {
	Name  string
	Token string `mask:"true"`
	Auth  AuthConfig
}

type AuthConfig struct {
	Id string
}

func main() {
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

	// Slice elements are addressed by index:
	//   CLIENTS_1_TOKEN=t9      overrides clients[1].token
	//   CLIENTS_2_NAME=baz      appends clients[2]
	//   CLIENTS_2_AUTH_ID=id-2  sets clients[2].auth.id
	//   PATHS_0=/one            builds paths, absent from application.yaml
	configJSON, err := adder.PrettyJSON(config)
	if err != nil {
		panic(err)
	}
	fmt.Println("Config loaded:")
	fmt.Println(configJSON)
}
