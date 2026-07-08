package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"janus/logger"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"

	"janus/types"
)

const (
	defaultAudience          = types.GCPTokenAudience
	googleCloudSDKAudience   = "32555940559.apps.googleusercontent.com"
	googleTokenEndpoint      = "https://oauth2.googleapis.com/token"
	metadataClientTimeout    = 3 * time.Second // Timeout for GCP metadata client requests
	identityTokenAudienceEnv = "IDENTITY_TOKEN_AUDIENCE"
	// Maximum length of an AWS STS RoleSessionName
	sessionIdentifierMaxLen = 64
)

// resolveAudience returns the identity token audience, honoring the
// IDENTITY_TOKEN_AUDIENCE environment variable if set. Note the audience only
// applies to GCE instance and service account tokens; the authorized_user flow
// always uses the Google Cloud SDK audience.
func resolveAudience() string {
	if audience := os.Getenv(identityTokenAudienceEnv); audience != "" {
		return audience
	}
	return defaultAudience
}

// credentialsFile represents the structure of the credentials JSON file
type credentialsFile struct {
	Type         string `json:"type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
}

// MetadataClient wraps the standard metadata.Client
type MetadataClient struct {
	*metadata.Client
}

// NewMetadataClient creates a new GCP metadata client
func NewMetadataClient() *MetadataClient {
	baseClient := metadata.NewClient(&http.Client{
		Timeout: metadataClientTimeout,
	})

	return &MetadataClient{
		Client: baseClient,
	}
}

// GetSessionIdentifier retrieves session identifier from command line flag, environment variable,
// or generates it from GCP metadata (in that order of precedence)
func GetSessionIdentifier(ctx context.Context, sessionIdFlag string, gcpMetadataClient *MetadataClient) (string, error) {
	// Check if provided via command line flag
	if sessionIdFlag != "" {
		return sessionIdFlag, nil
	}

	// Check if provided via environment variable
	if envSessionId := os.Getenv(types.EnvSessionID); envSessionId != "" {
		return envSessionId, nil
	}

	// Try creating it from GCP metadata
	logger.Logger.Debug("Attempting to create session identifier from GCP metadata")

	sessionId, err := CreateSessionIdentifier(ctx, gcpMetadataClient)
	if err == nil {
		return sessionId, nil
	}

	// Don't fall back to the local hostname when the metadata lookup failed
	// because the context was cancelled or timed out
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}

	// Fall back to local hostname if GCP metadata fails
	logger.Logger.Debug("Failed to create session identifier from GCP metadata, falling back to OS hostname", "error", err)
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("couldn't determine session identifier: %w", err)
	}

	// Use hostname as fallback
	logger.Logger.Debug("Using local hostname as session identifier", "hostname", hostname)
	return hostname, nil
}

// CreateSessionIdentifier constructs AWS session identifier from GCP metadata information.
// This implementation uses concatenation of GCP project ID and machine hostname.
func CreateSessionIdentifier(ctx context.Context, c *MetadataClient) (string, error) {
	projectID, err := c.ProjectIDWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch ProjectID from GCP metadata server: %w", err)
	}

	hostname, err := c.HostnameWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch Hostname from GCP metadata server: %w", err)
	}

	s := fmt.Sprintf("%s-%s", projectID, hostname)
	if len(s) > sessionIdentifierMaxLen {
		s = s[:sessionIdentifierMaxLen]
	}
	return s, nil
}

// printIdentityTokenIfEnabled prints the identity token if enabled in config and log level is DEBUG
func printIdentityTokenIfEnabled(config types.Config, tokenSource oauth2.TokenSource) {
	if config.PrintIdToken && strings.EqualFold(config.LogLevel, "DEBUG") {
		if token, err := tokenSource.Token(); err != nil {
			logger.Logger.Error(fmt.Errorf("failed to get identity token for printing: %w", err).Error())
		} else {
			logger.Logger.Debug("Google identity token", "id_token", token.AccessToken)
		}
	}
}

// TokenSource returns an OAuth2 token source for authenticating with GCP.
// It first tries to get a token from GCE metadata if running on GCP,
// then falls back to local credentials if not on GCP.
func TokenSource(ctx context.Context, config types.Config) (oauth2.TokenSource, error) {
	// First try GCE metadata if running on GCP
	if metadata.OnGCE() {
		token, err := fetchInstanceIdentityToken(ctx)
		if err == nil {
			tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: token,
			})
			printIdentityTokenIfEnabled(config, tokenSource)
			return tokenSource, nil
		}
		// Log the error but continue to try other methods
		logger.Logger.Debug("Failed to get GCE instance token", "error", err)
	}

	// Try generating token from local credentials
	token, err := generateIdentityToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity token: %w", err)
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	printIdentityTokenIfEnabled(config, tokenSource)
	return tokenSource, nil
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

// fetchInstanceIdentityToken retrieves an identity token from GCE metadata
func fetchInstanceIdentityToken(ctx context.Context) (string, error) {
	audience := resolveAudience()

	// Use the built-in Google SDK function to get an identity token
	idTokenSource, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		return "", fmt.Errorf("failed to create identity token source: %w", err)
	}

	// Get the token from the source
	token, err := idTokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get instance identity token: %w", err)
	}

	return token.AccessToken, nil
}

// generateIdentityToken generates an identity token from local credentials
func generateIdentityToken(ctx context.Context) (string, error) {
	audience := resolveAudience()

	creds, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get default credentials: %w", err)
	}

	if len(creds.JSON) == 0 {
		return "", fmt.Errorf("application default credentials contain no key file; set GOOGLE_APPLICATION_CREDENTIALS to a service account or authorized user key file")
	}

	// Parse the credentials to determine the type
	var cf credentialsFile
	if err := json.Unmarshal(creds.JSON, &cf); err != nil {
		return "", fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Handle authorized user credentials
	if cf.Type == "authorized_user" {
		// Exchange refresh token for ID token
		data := url.Values{}
		data.Set("client_id", cf.ClientID)
		data.Set("client_secret", cf.ClientSecret)
		data.Set("refresh_token", cf.RefreshToken)
		data.Set("grant_type", "refresh_token")
		data.Set("audience", googleCloudSDKAudience)

		req, err := http.NewRequestWithContext(ctx, "POST", googleTokenEndpoint, strings.NewReader(data.Encode()))
		if err != nil {
			return "", fmt.Errorf("failed to create token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to execute token request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read token response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
		}

		var tokenResp struct {
			IDToken string `json:"id_token"`
		}
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return "", fmt.Errorf("failed to parse token response: %w", err)
		}

		return tokenResp.IDToken, nil
	}

	// Handle service account credentials
	if cf.Type == "service_account" {
		// Use Google's idtoken package to create a properly formatted OIDC token
		tokenSource, err := idtoken.NewTokenSource(ctx, audience, idtoken.WithCredentialsJSON(creds.JSON))
		if err != nil {
			return "", fmt.Errorf("failed to create token source: %w", err)
		}

		token, err := tokenSource.Token()
		if err != nil {
			return "", fmt.Errorf("failed to generate token: %w", err)
		}

		return token.AccessToken, nil
	}

	return "", fmt.Errorf("unsupported credential type: %s", cf.Type)
}
