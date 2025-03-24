package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mds "github.com/salrashid123/gce_metadata_server"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"janus/gcp"
	"janus/types"
)

var (
	gceInstanceHostname = "janus-go-instance-hostname"
	gcpProjectID        = "janus-go"
	credsTokenSource    = oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken:  "mock_token",
			RefreshToken: "mock_refresh",
		},
	)
)

// MockGCPMetadataServer creates and returns a mock GCP metadata server.
// It uses default credentials for the metadata server and binds to the
// local interface on port 8080. The server is configured with the provided
// GCP project ID and GCE instance hostname.
func MockGCPMetadataServer(tokenSource *oauth2.TokenSource) *mds.MetadataServer {
	ctx := context.Background()

	creds := &google.Credentials{}

	if tokenSource == nil {
		// Use default credentials for the metadata server if none are provided to the function
		creds, _ = google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	} else {
		creds = &google.Credentials{
			ProjectID:   gcpProjectID,
			TokenSource: *tokenSource,
		}
	}

	serverConfig := &mds.ServerConfig{
		BindInterface: "127.0.0.1",
		Port:          ":8080",
	}

	claims := &mds.Claims{
		ComputeMetadata: mds.ComputeMetadata{
			V1: mds.V1{
				Project: mds.Project{
					ProjectID: gcpProjectID,
				},
				Instance: mds.Instance{
					Hostname: gceInstanceHostname,
				},
			},
		},
	}
	f, err := mds.NewMetadataServer(ctx, serverConfig, creds, claims)
	if err != nil {
		return nil
	}
	return f
}

// TestCreateSessionIdentifier is a unit test function that tests the CreateSessionIdentifier function.
// It creates a mock metadata server, sets the local metadata server, starts the mock metadata server,
// and then calls the CreateSessionIdentifier function to generate a session ID.
func TestCreateSessionIdentifier(t *testing.T) {
	// Create a mock metadata server
	f := MockGCPMetadataServer(&credsTokenSource)

	// Use local metadata server
	t.Setenv("GCE_METADATA_HOST", "127.0.0.1:8080")

	// Start a mock metadata server
	err := f.Start()
	if err != nil {
		t.Errorf("Failed to start metadata server: %v", err)
	}
	defer f.Shutdown()

	// Create a GCP metadata client
	client := gcp.NewMetadataClient()

	sessionID, err := gcp.CreateSessionIdentifier(client)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedSessionID := fmt.Sprintf("%s-%s", gcpProjectID, gceInstanceHostname)[:32]
	if sessionID != expectedSessionID {
		t.Errorf("Unexpected session ID: got %s, want %s", sessionID, expectedSessionID)
	}
}

// TestGCEMatadataDataHostname is a unit test function that tests the functionality
// of fetching the hostname from the metadata server.
func TestGCEMatadataDataHostname(t *testing.T) {
	// Start a mock metadata server
	f := MockGCPMetadataServer(&credsTokenSource)

	// Use local metadata server
	t.Setenv("GCE_METADATA_HOST", "127.0.0.1:8080")

	err := f.Start()
	if err != nil {
		t.Errorf("Failed to start metadata server: %v", err)
	}
	defer f.Shutdown()

	// Create a GCP metadata client
	client := gcp.NewMetadataClient()

	if client == nil {
		t.Errorf("Expected non-nil metadata client, got nil")
	}

	// Test the metadata client by fetching the hostname
	metadataHostname, err := client.Hostname()
	if err != nil {
		t.Errorf("Failed to fetch hostname from metadata server: %v", err)
	}

	assert.Equal(t, gceInstanceHostname, metadataHostname, "Retrieved hostname does not match expected value")
}

// TestGCEMatadataDataProjectID tests fetching the project ID from metadata server
func TestGCEMatadataDataProjectID(t *testing.T) {
	// Start a mock metadata server
	f := MockGCPMetadataServer(&credsTokenSource)

	// Use local metadata server
	t.Setenv("GCE_METADATA_HOST", "127.0.0.1:8080")

	err := f.Start()
	if err != nil {
		t.Errorf("Failed to start metadata server: %v", err)
	}
	defer f.Shutdown()

	// Create a GCP metadata client
	client := gcp.NewMetadataClient()

	if client == nil {
		t.Errorf("Expected non-nil metadata client, got nil")
	}

	// Test the metadata client by fetching the project ID
	metadataProjectID, err := client.ProjectID()
	if err != nil {
		t.Errorf("Failed to fetch hostname from metadata server: %v", err)
	}

	assert.Equal(t, gcpProjectID, metadataProjectID, "Retrieved project ID does not match expected value")
}

// TestAWSTempCredentials tests the marshaling and unmarshaling of AWSTempCredentials
func TestAWSTempCredentials(t *testing.T) {
	// Create a sample AWSTempCredentials instance
	credentials := types.AWSTempCredentials{
		Version:         1,
		AccessKeyId:     "access_key",
		SecretAccessKey: "secret_key",
		SessionToken:    "session_token",
		Expiration:      time.Now(),
	}

	// Convert the credentials to JSON
	jsonData, err := json.Marshal(credentials)
	if err != nil {
		t.Errorf("Failed to marshal AWSTempCredentials to JSON: %v", err)
	}

	// Convert the JSON back to AWSTempCredentials
	var parsedCredentials types.AWSTempCredentials
	err = json.Unmarshal(jsonData, &parsedCredentials)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON to AWSTempCredentials: %v", err)
	}

	// Compare the original and parsed credentials
	assert.Equal(t, credentials.Version, parsedCredentials.Version, "Version mismatch")
	assert.Equal(t, credentials.AccessKeyId, parsedCredentials.AccessKeyId, "AccessKeyId mismatch")
	assert.Equal(t, credentials.SecretAccessKey, parsedCredentials.SecretAccessKey, "SecretAccessKey mismatch")
	assert.Equal(t, credentials.SessionToken, parsedCredentials.SessionToken, "SessionToken mismatch")
	assert.True(t, credentials.Expiration.Equal(parsedCredentials.Expiration), "Expiration mismatch")
}
