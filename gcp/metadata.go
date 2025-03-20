package gcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"

	"janus/types"
)

// NewMetadataClient creates a new GCP metadata client
func NewMetadataClient() *metadata.Client {
	return metadata.NewClient(&http.Client{Timeout: 1 * time.Second})
}

// CreateSessionIdentifier constructs AWS session identifier from GCP metadata information.
// This implementation uses concatenation of GCP project ID and machine hostname.
func CreateSessionIdentifier(c *metadata.Client) (string, error) {
	ctx := context.Background()
	projectId, err := c.ProjectIDWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch ProjectId from GCP metadata server: %w", err)
	}

	hostname, err := c.HostnameWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch Hostname from GCP metadata server: %w", err)
	}

	return fmt.Sprintf("%s-%s", projectId, hostname)[:32], nil
}

// TokenSource returns an OAuth2 token source for authenticating with GCP GKE.
// It fetches the GCP default credentials from the environment and uses them to obtain an identity token.
func TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	credentials, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch GCP default credentials from environment: %w", err)
	}

	ts, err := idtoken.NewTokenSource(ctx, types.GCPTokenAudience, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch GCP identity token: %w", err)
	}
	return ts, nil
}

// CustomIdentityTokenRetriever implements the identity token retrieval functionality
type CustomIdentityTokenRetriever struct {
	TokenSource oauth2.TokenSource
}

// GetIdentityToken retrieves the identity token from the token source
func (obj CustomIdentityTokenRetriever) GetIdentityToken() ([]byte, error) {
	token, err := obj.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve identity token: %w", err)
	}
	return []byte(token.AccessToken), nil
}
