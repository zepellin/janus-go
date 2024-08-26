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

var (
	logger             = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	GCP_TOKEN_AUDIENCE = "gcp"
)

// Creates GCP metadata client
func gcpMetadataClient() *metadata.Client {
	c := metadata.NewClient(&http.Client{Timeout: 1 * time.Second})
	return c
}

// Constucts AWs session identifier from GCP metadata infrmation.
// This implementation uses concentration of  GCP project ID and machine hostname
func createSessionIdentifier(c *metadata.Client) (string, error) {
	projectId, err := c.ProjectID()
	if err != nil {
		logger.Error("Couldn't fetch ProjectId from GCP metadata server")
		return "", err
	}

	hostname, err := c.Hostname()
	if err != nil {
		logger.Error("Couldn't fetch Hostname from GCP metadata server")
		return "", err
	}

	return (fmt.Sprintf("%s-%s", projectId, hostname)[:32]), nil
}

// gcpTokenSource returns an OAuth2 token source for authenticating with GCP GKE.
// It fetches the GCP default credentials from the environment and uses them to obtain an identity token.
func gcpTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	credentials, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		logger.Debug("Couldn't fetch GCP default credentials from environment")
		return nil, err
	}

	ts, err := idtoken.NewTokenSource(ctx, GCP_TOKEN_AUDIENCE, option.WithCredentials(credentials))
	if err != nil {
		logger.Debug("Couldn't fetch GCP identity token")
		return nil, err
	}
	return ts, nil
}

type customIdentityTokenRetriever struct {
	tokenSource oauth2.TokenSource
}

func (obj customIdentityTokenRetriever) GetIdentityToken() ([]byte, error) {
	token, err := obj.tokenSource.Token()
	if err != nil {
		return nil, err
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

// Custom stringer for awsCredentials stuct
func (c awsTempCredentials) String() string {
	return fmt.Sprintf("{\"Version\": %d, \"AccessKeyId\": \"%s\", \"SecretAccessKey\": \"%s\", \"SessionToken\": \"%s\", \"Expiration\": \"%s\"}", 1, c.AccessKeyId, c.SecretAccessKey, c.SessionToken, c.Expiration.Format("2006-01-02T15:04:05-07:00"))
}

func main() {
	awsAssumeRoleArn := flag.String("rolearn", "", "AWS role ARN to assume (required)")
	stsRegion := flag.String("stsregion", "us-east-1", "AWS STS region to which requests are made (optional)")

	flag.Parse()
	if *awsAssumeRoleArn == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	gcpMetadataClient := gcpMetadataClient()
	if gcpMetadataClient == nil {
		logger.Error("Expected non-nil metadata client, got nil")
		os.Exit(1)
	}

	sessionIdentifier, err := createSessionIdentifier(gcpMetadataClient)
	if err != nil {
		logger.Error("Failed to create session identifier from GCP metadata, %s" + err.Error())
		os.Exit(1)
	}

	assumeRoleCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*stsRegion))
	if err != nil {
		logger.Error("failed to load default AWS config, %s" + err.Error())
		os.Exit(1)
	}

	gcpMetadataTokenSource, err := gcpTokenSource(ctx)
	if err != nil {
		logger.Error("Failed to get JWT token from GCP metadata, %s" + err.Error())
		os.Exit(1)
	}

	gcpMetadataToken := customIdentityTokenRetriever{tokenSource: gcpMetadataTokenSource}

	stsAssumeClient := sts.NewFromConfig(assumeRoleCfg)
	awsCredsCache := aws.NewCredentialsCache(stscreds.NewWebIdentityRoleProvider(
		stsAssumeClient,
		*awsAssumeRoleArn,
		gcpMetadataToken,
		func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = sessionIdentifier
		}),
	)

	awsCredentials, err := awsCredsCache.Retrieve(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("Couldn't retrieve AWS credentials %s", err))
		os.Exit(1)
	}

	credentials := awsTempCredentials{
		Version:         1,
		AccessKeyId:     awsCredentials.AccessKeyID,
		SecretAccessKey: awsCredentials.SecretAccessKey,
		SessionToken:    awsCredentials.SessionToken,
		Expiration:      awsCredentials.Expires}

	fmt.Printf("%+v\n", credentials)
}
