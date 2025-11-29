package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port string
	}
	Database struct {
		DSN string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	JWT struct {
		Secret string
	}
}

var C Config

func Load() {
	viper.SetConfigFile("config/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	if err := viper.Unmarshal(&C); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
}
