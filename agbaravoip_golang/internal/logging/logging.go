package logging

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/user/agbaravoip_golang/internal/config"
)

// Logger is a global instance, but InitLogger will return a configured instance.
// This global can be used by other packages if they import logging.
var Logger = logrus.New()

// InitLogger initializes the logger based on the configuration and returns the instance.
func InitLogger(cfg config.Config) *logrus.Logger {
	// Use the global Logger instance for configuration
	Logger.SetOutput(os.Stdout) // Or a file, etc.
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		Logger.SetLevel(logrus.InfoLevel) // Default level if parse fails
		Logger.Warnf("Failed to parse log level: %s, defaulting to info", cfg.LogLevel)
	} else {
		Logger.SetLevel(level)
	}

	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return Logger // Return the configured global instance
}
