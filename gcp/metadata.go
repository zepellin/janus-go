package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"

	"janus/types"
)

const (
	defaultAudience        = types.GCPTokenAudience
	googleCloudSDKAudience = "32555940559.apps.googleusercontent.com"
	googleTokenInfoURL     = "https://oauth2.googleapis.com/token"
)

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

// TokenSource returns an OAuth2 token source for authenticating with GCP.
// It first tries to get a token from GCE metadata if running on GCP,
// then falls back to local credentials if not on GCP.
func TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	// First try GCE metadata if running on GCP
	if metadata.OnGCE() {
		token, err := fetchInstanceIdentityToken(ctx)
		if err == nil {
			return oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: token,
			}), nil
		}
		// Log the error but continue to try other methods
		fmt.Printf("Failed to get GCE instance token: %v\n", err)
	}

	// Try generating token from local credentials
	token, err := generateIdentityToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity token: %w", err)
	}

	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	}), nil
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
	audience := os.Getenv("IDENTITY_TOKEN_AUDIENCE")
	if audience == "" {
		audience = defaultAudience
	}

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
	audience := os.Getenv("IDENTITY_TOKEN_AUDIENCE")
	if audience == "" {
		audience = defaultAudience
	}

	creds, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get default credentials: %w", err)
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

		req, err := http.NewRequestWithContext(ctx, "POST", googleTokenInfoURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create token request: %w", err)
		}
		req.URL.RawQuery = data.Encode()
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
		conf, err := google.JWTConfigFromJSON(creds.JSON)
		if err != nil {
			return "", fmt.Errorf("failed to parse JWT config: %w", err)
		}

		now := time.Now()
		claims := jwt.MapClaims{
			"iss": conf.Email,
			"sub": conf.Email,
			"aud": audience,
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(cf.PrivateKey))
		if err != nil {
			return "", fmt.Errorf("failed to parse private key: %w", err)
		}

		signedToken, err := token.SignedString(key)
		if err != nil {
			return "", fmt.Errorf("failed to sign token: %w", err)
		}

		return signedToken, nil
	}

	return "", fmt.Errorf("unsupported credential type: %s", cf.Type)
}
