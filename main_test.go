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
	"janus/logger"
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

func init() {
	// Initialize logger for tests
	logger.InitLogger("ERROR")
}

// MockGCPMetadataServer creates and returns a mock GCP metadata server.
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

// setupMockServer is a helper function that sets up and verifies the mock metadata server is ready
func setupMockServer(t *testing.T) (*mds.MetadataServer, func()) {
	f := MockGCPMetadataServer(&credsTokenSource)
	t.Setenv("GCE_METADATA_HOST", "127.0.0.1:8080")

	err := f.Start()
	if err != nil {
		t.Fatalf("Failed to start metadata server: %v", err)
	}

	// Give the server a moment to start up
	time.Sleep(200 * time.Millisecond)

	return f, func() {
		f.Shutdown()
	}
}

// TestCreateSessionIdentifier is a unit test function that tests the CreateSessionIdentifier function.
func TestCreateSessionIdentifier(t *testing.T) {
	_, cleanup := setupMockServer(t)
	defer cleanup()

	ctx := context.Background()
	client := gcp.NewMetadataClient(ctx)

	sessionID, err := gcp.CreateSessionIdentifier(ctx, client)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedSessionID := fmt.Sprintf("%s-%s", gcpProjectID, gceInstanceHostname)[:32]
	if sessionID != expectedSessionID {
		t.Errorf("Unexpected session ID: got %s, want %s", sessionID, expectedSessionID)
	}
}

func TestGCEMetadataHostname(t *testing.T) {
	_, cleanup := setupMockServer(t)
	defer cleanup()

	ctx := context.Background()
	client := gcp.NewMetadataClient(ctx)

	if client == nil {
		t.Errorf("Expected non-nil metadata client, got nil")
	}

	metadataHostname, err := client.Hostname()
	if err != nil {
		t.Errorf("Failed to fetch hostname from metadata server: %v", err)
	}

	assert.Equal(t, gceInstanceHostname, metadataHostname, "Retrieved hostname does not match expected value")
}

func TestGCEMetadataProjectID(t *testing.T) {
	_, cleanup := setupMockServer(t)
	defer cleanup()

	ctx := context.Background()
	client := gcp.NewMetadataClient(ctx)

	if client == nil {
		t.Errorf("Expected non-nil metadata client, got nil")
	}

	metadataProjectID, err := client.ProjectID()
	if err != nil {
		t.Errorf("Failed to fetch project ID from metadata server: %v", err)
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

// TestContextCancellation verifies that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	_, cleanup := setupMockServer(t)
	defer cleanup()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create a GCP metadata client with the cancellable context
	client := gcp.NewMetadataClient(ctx)

	// Cancel the context before making the request
	cancel()

	// Sleep briefly to ensure cancellation propagates
	time.Sleep(10 * time.Millisecond)

	// Try to get ProjectID directly first
	_, err := client.ProjectIDWithContext(ctx)
	if err == nil {
		t.Error("Expected error when getting ProjectID with cancelled context, got nil")
	} else if ctx.Err() != err {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}

	// Now try to get session identifier
	_, err = gcp.GetSessionIdentifier(ctx, "", client)
	if err == nil {
		t.Error("Expected error when getting session identifier with cancelled context, got nil")
	} else if ctx.Err() != err {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestContextTimeout verifies that operations respect context timeout
func TestContextTimeout(t *testing.T) {
	_, cleanup := setupMockServer(t)
	defer cleanup()

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Create a GCP metadata client with the timeout context
	client := gcp.NewMetadataClient(ctx)

	// Sleep to ensure timeout occurs
	time.Sleep(10 * time.Millisecond)

	// Try to get ProjectID directly first
	_, err := client.ProjectIDWithContext(ctx)
	if err == nil {
		t.Error("Expected error when getting ProjectID with timed out context, got nil")
	} else if ctx.Err() != err {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}

	// Now try to get session identifier
	_, err = gcp.GetSessionIdentifier(ctx, "", client)
	if err == nil {
		t.Error("Expected error when getting session identifier with timed out context, got nil")
	} else if ctx.Err() != err {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}
