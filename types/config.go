package types

// Config holds the configuration settings
type Config struct {
	// PrintIdToken indicates whether to print the identity token when log level is DEBUG
	PrintIdToken bool
	// LogLevel specifies the logging level (DEBUG, INFO, WARN, ERROR)
	LogLevel string
}
