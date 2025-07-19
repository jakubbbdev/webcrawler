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
	// Standardwerte setzen
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("TIMEOUT", 30)

	// Umgebungsvariablen lesen
	viper.AutomaticEnv()

	// Konfigurationsdatei lesen (falls vorhanden)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		// Konfigurationsdatei ist optional
		fmt.Printf("Keine Konfigurationsdatei gefunden: %v\n", err)
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		panic(fmt.Sprintf("Konfiguration konnte nicht geladen werden: %v", err))
	}

	// Port aus Umgebungsvariable überschreiben
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	return config
} 