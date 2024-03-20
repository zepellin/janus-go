package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAWSTempCredentials(t *testing.T) {
	// Create a sample awsTempCredentials instance
	credentials := awsTempCredentials{
		Version:         1,
		AccessKeyId:     "access_key",
		SecretAccessKey: "secret_key",
		SessionToken:    "session_token",
		Expiration:      time.Now(),
	}

	// Convert the credentials to JSON
	jsonData, err := json.Marshal(credentials)
	if err != nil {
		t.Errorf("Failed to marshal awsTempCredentials to JSON: %v", err)
	}

	// Convert the JSON back to awsTempCredentials
	var parsedCredentials awsTempCredentials
	err = json.Unmarshal(jsonData, &parsedCredentials)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON to awsTempCredentials: %v", err)
	}

	// Compare the original and parsed credentials
	if credentials.Version != parsedCredentials.Version {
		t.Errorf("Version mismatch: expected %d, got %d", credentials.Version, parsedCredentials.Version)
	}
	if credentials.AccessKeyId != parsedCredentials.AccessKeyId {
		t.Errorf("AccessKeyId mismatch: expected %s, got %s", credentials.AccessKeyId, parsedCredentials.AccessKeyId)
	}
	if credentials.SecretAccessKey != parsedCredentials.SecretAccessKey {
		t.Errorf("SecretAccessKey mismatch: expected %s, got %s", credentials.SecretAccessKey, parsedCredentials.SecretAccessKey)
	}
	if credentials.SessionToken != parsedCredentials.SessionToken {
		t.Errorf("SessionToken mismatch: expected %s, got %s", credentials.SessionToken, parsedCredentials.SessionToken)
	}
	if !credentials.Expiration.Equal(parsedCredentials.Expiration) {
		t.Errorf("Expiration mismatch: expected %v, got %v", credentials.Expiration, parsedCredentials.Expiration)
	}
}
