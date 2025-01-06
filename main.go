package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	gcpTokenAudience = "gcp"
	stsRegionDefault = "us-east-1"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Creates GCP metadata client
func newGCPMetadataClient() *metadata.Client {
	return metadata.NewClient(&http.Client{Timeout: 1 * time.Second})
}

// Constructs AWS session identifier from GCP metadata information.
// This implementation uses concatenation of GCP project ID and machine hostname.
func createSessionIdentifier(c *metadata.Client) (string, error) {
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

// gcpTokenSource returns an OAuth2 token source for authenticating with GCP GKE.
// It fetches the GCP default credentials from the environment and uses them to obtain an identity token.
func gcpTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	credentials, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch GCP default credentials from environment: %w", err)
	}

	ts, err := idtoken.NewTokenSource(ctx, gcpTokenAudience, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch GCP identity token: %w", err)
	}
	return ts, nil
}

type customIdentityTokenRetriever struct {
	tokenSource oauth2.TokenSource
}

func (obj customIdentityTokenRetriever) GetIdentityToken() ([]byte, error) {
	token, err := obj.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve identity token: %w", err)
	}
	return []byte(token.AccessToken), nil
}

type awsTempCredentials struct {
	Version         int       `json:"Version"`
	AccessKeyId     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

// Custom stringer for awsTempCredentials struct
func (c awsTempCredentials) String() string {
	return fmt.Sprintf("{\"Version\": %d, \"AccessKeyId\": \"%s\", \"SecretAccessKey\": \"%s\", \"SessionToken\": \"%s\", \"Expiration\": \"%s\"}", 1, c.AccessKeyId, c.SecretAccessKey, c.SessionToken, c.Expiration.Format(time.RFC3339))
}

func main() {
	awsAssumeRoleArn := flag.String("rolearn", "", "AWS role ARN to assume (required)")
	stsRegion := flag.String("stsregion", stsRegionDefault, "AWS STS region to which requests are made (optional)")

	flag.Parse()
	if *awsAssumeRoleArn == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	gcpMetadataClient := newGCPMetadataClient()
	if gcpMetadataClient == nil {
		logger.Error("Failed to create GCP metadata client: got nil")
		os.Exit(1)
	}

	sessionIdentifier, err := createSessionIdentifier(gcpMetadataClient)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create session identifier: %w", err).Error())
		os.Exit(1)
	}

	// Ensure stsRegion isn't nil
	if stsRegion == nil {
		logger.Error("stsRegion is nil, cannot proceed")
		os.Exit(1)
	}

	assumeRoleCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*stsRegion))
	if err != nil {
		logger.Error(fmt.Errorf("failed to load AWS config: %w", err).Error())
		os.Exit(1)
	}

	gcpMetadataTokenSource, err := gcpTokenSource(ctx)
	if err != nil {
		logger.Error(fmt.Errorf("failed to retrieve GCP identity token: %w", err).Error())
		os.Exit(1)
	}

	gcpMetadataToken := customIdentityTokenRetriever{tokenSource: gcpMetadataTokenSource}

	stsAssumeClient := sts.NewFromConfig(assumeRoleCfg)
	awsCredsCache := aws.NewCredentialsCache(
		stscreds.NewWebIdentityRoleProvider(
			stsAssumeClient,
			*awsAssumeRoleArn,
			gcpMetadataToken,
			func(o *stscreds.WebIdentityRoleOptions) {
				o.RoleSessionName = sessionIdentifier
			},
		),
	)

	awsCredentials, err := awsCredsCache.Retrieve(ctx)
	if err != nil {
		logger.Error(fmt.Errorf("failed to retrieve AWS credentials: %w", err).Error())
		os.Exit(1)
	}

	credentials := awsTempCredentials{
		Version:         1,
		AccessKeyId:     awsCredentials.AccessKeyID,
		SecretAccessKey: awsCredentials.SecretAccessKey,
		SessionToken:    awsCredentials.SessionToken,
		Expiration:      awsCredentials.Expires,
	}

	fmt.Printf("%+v\n", credentials)
}
