package types

import (
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
