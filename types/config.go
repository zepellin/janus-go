package types

// Config holds the global configuration settings
type Config struct {
	// PrintIdToken indicates whether to print the identity token when log level is DEBUG
	PrintIdToken bool
	// LogLevel specifies the logging level (DEBUG, INFO, WARN, ERROR)
	LogLevel string
}

var globalConfig Config

// SetConfig sets the global configuration
func SetConfig(config Config) {
	globalConfig = config
}

// GetConfig returns the global configuration
func GetConfig() Config {
	return globalConfig
}
