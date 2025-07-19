package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Port     int    `mapstructure:"PORT"`
	LogLevel string `mapstructure:"LOG_LEVEL"`
	Timeout  int    `mapstructure:"TIMEOUT"`
}

func Load() *Config {
	// Set default values
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("TIMEOUT", 30)

	// Read environment variables
	viper.AutomaticEnv()

	// Read configuration file (if present)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		// Configuration file is optional
		fmt.Printf("No configuration file found: %v\n", err)
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		panic(fmt.Sprintf("Configuration could not be loaded: %v", err))
	}

	// Override port from environment variable
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	return config
}
