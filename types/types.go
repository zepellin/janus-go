package types

import (
	"fmt"
	"time"
)

const (
	GCPTokenAudience = "gcp"
	STSRegionDefault = "us-east-1"
	EnvSessionID     = "AWS_SESSION_IDENTIFIER" // Environment variable name for session identifier
)

// AWSTempCredentials represents temporary AWS credentials
type AWSTempCredentials struct {
	Version         int       `json:"Version"`
	AccessKeyId     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

// String returns a JSON representation of AWSTempCredentials
func (c AWSTempCredentials) String() string {
	return fmt.Sprintf("{\"Version\": %d, \"AccessKeyId\": \"%s\", \"SecretAccessKey\": \"%s\", \"SessionToken\": \"%s\", \"Expiration\": \"%s\"}",
		1, c.AccessKeyId, c.SecretAccessKey, c.SessionToken, c.Expiration.Format(time.RFC3339))
}
