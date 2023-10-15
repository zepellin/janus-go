package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

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

// Retrieves GCE identity token (JWT) and retuens [customIdentityTokenRetriever] instance
// containing the token. This is to be then used in [stscreds.NewWebIdentityRoleProvider]
// function.
func gcpRetrieveGCEVMToken(ctx context.Context) (customIdentityTokenRetriever, error) {
	url := "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?format=full&audience=gcp"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return customIdentityTokenRetriever{token: nil}, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return customIdentityTokenRetriever{token: nil}, fmt.Errorf("client.Do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return customIdentityTokenRetriever{token: nil}, fmt.Errorf("status code %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return customIdentityTokenRetriever{token: nil}, fmt.Errorf("io.ReadAll: %w", err)
	}
	gcpMetadataToken := customIdentityTokenRetriever{token: b}
	return gcpMetadataToken, nil
}

type customIdentityTokenRetriever struct {
	token []byte
}

func (obj customIdentityTokenRetriever) GetIdentityToken() ([]byte, error) {
	return obj.token, nil
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

	sessionIdentifier, err := createSessionIdentifier(gcpMetadataClient())
	if err != nil {
		logger.Error("Failed to create session identifier from GCP metadata, %s" + err.Error())
		os.Exit(1)
	}

	assumeRoleCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*stsRegion))
	if err != nil {
		logger.Error("failed to load default AWS config, %s" + err.Error())
		os.Exit(1)
	}

	gcpMetadataToken, err := gcpRetrieveGCEVMToken(ctx)
	if err != nil {
		logger.Error("Failed to get JWT token from GCP metadata, %s" + err.Error())
		os.Exit(1)
	}

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
		logger.Error("Couldn't retrieve AWS credentials %s", err)
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
