package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

const (
	defaultAudience    = "32555940559.apps.googleusercontent.com" // Default gcloud audience
	googleTokenInfoURL = "https://oauth2.googleapis.com/token"
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

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// First check if running on GCE
	if metadata.OnGCE() {
		token, err := fetchInstanceIdentityToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch instance identity token: %v", err)
		}
		fmt.Println(token)
		return nil
	}

	// If not on GCE, generate token from default credentials
	token, err := generateIdentityToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate identity token: %v", err)
	}

	fmt.Println(token)
	return nil
}

func fetchInstanceIdentityToken(ctx context.Context) (string, error) {
	audience := os.Getenv("IDENTITY_TOKEN_AUDIENCE")
	if audience == "" {
		audience = defaultAudience
	}

	client, err := idtoken.NewClient(ctx, audience)
	if err != nil {
		return "", fmt.Errorf("failed to create identity token client: %v", err)
	}

	token, err := client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	return token.AccessToken, nil
}

func generateIdentityToken(ctx context.Context) (string, error) {
	audience := os.Getenv("IDENTITY_TOKEN_AUDIENCE")
	if audience == "" {
		audience = defaultAudience
	}

	creds, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get default credentials: %v", err)
	}

	// Parse the credentials to determine the type
	var cf credentialsFile
	if err := json.Unmarshal(creds.JSON, &cf); err != nil {
		return "", fmt.Errorf("failed to parse credentials: %v", err)
	}

	// Handle authorized user credentials
	if cf.Type == "authorized_user" {
		// Exchange refresh token for ID token
		data := url.Values{}
		data.Set("client_id", cf.ClientID)
		data.Set("client_secret", cf.ClientSecret)
		data.Set("refresh_token", cf.RefreshToken)
		data.Set("grant_type", "refresh_token")
		data.Set("audience", audience)

		req, err := http.NewRequestWithContext(ctx, "POST", googleTokenInfoURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create token request: %v", err)
		}
		req.URL.RawQuery = data.Encode()
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to execute token request: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read token response: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
		}

		var tokenResp struct {
			IDToken string `json:"id_token"`
		}
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return "", fmt.Errorf("failed to parse token response: %v", err)
		}

		return tokenResp.IDToken, nil
	}

	// Handle service account credentials
	if cf.Type == "service_account" {
		conf, err := google.JWTConfigFromJSON(creds.JSON)
		if err != nil {
			return "", fmt.Errorf("failed to parse JWT config: %v", err)
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
		key, err := jwt.ParseRSAPrivateKeyFromPEM(conf.PrivateKey)
		if err != nil {
			return "", fmt.Errorf("failed to parse private key: %v", err)
		}

		signedToken, err := token.SignedString(key)
		if err != nil {
			return "", fmt.Errorf("failed to sign token: %v", err)
		}

		return signedToken, nil
	}

	return "", fmt.Errorf("unsupported credential type: %s", cf.Type)
}
