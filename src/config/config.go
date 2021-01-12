package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// Config holds configuration of this tool
type Config struct {
	Token string `required:"true"`
}

// Load returns configuration taken from environment variables.
func Load() *Config {
	var config Config

	err := envconfig.Process("rosti", &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	return &config
}
