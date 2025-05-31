package config

import (
	"strings"
	"github.com/spf13/viper"
)

// FreeswitchConfig stores Freeswitch connection details.
type FreeswitchConfig struct {
	FSAddress             string `mapstructure:"FS_ADDRESS"`
	FSPort                string `mapstructure:"FS_PORT"`
	FSPassword            string `mapstructure:"FS_PASSWORD"`
	FSOutboundListenAddress string `mapstructure:"FS_OUTBOUND_LISTEN_ADDRESS"` // e.g., ":8084" or "0.0.0.0:8084"
}

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variable.
type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
	LogLevel   string `mapstructure:"LOG_LEVEL"`

	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSchema   string `mapstructure:"DB_SCHEMA"` // e.g. "public"

	Freeswitch FreeswitchConfig `mapstructure:",squash"` // Embed FreeswitchConfig
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config") // Register config file name (no extension)
	viper.SetConfigType("yml")   // Look for specific type

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")

	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "agbarauser")
	viper.SetDefault("DB_PASSWORD", "agbarapassword")
	viper.SetDefault("DB_NAME", "agbaravoip")
	viper.SetDefault("DB_SCHEMA", "public")

	viper.SetDefault("FS_ADDRESS", "localhost")
	viper.SetDefault("FS_PORT", "8021")
	viper.SetDefault("FS_PASSWORD", "ClueCon")
	viper.SetDefault("FS_OUTBOUND_LISTEN_ADDRESS", ":8084") // Default port for ESL Outbound Server


	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return
		}
		// Config file not found; ignore error and use defaults/env vars
	}

	err = viper.Unmarshal(&config)
	return
}
