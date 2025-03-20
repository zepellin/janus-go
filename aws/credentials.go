package aws

import (
	"context"
	"fmt"
	"os"

	"log/slog"

	"cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"janus/gcp"
	"janus/types"
)

// GetSessionIdentifier retrieves session identifier from command line flag, environment variable,
// or generates it from GCP metadata (in that order of precedence)
func GetSessionIdentifier(sessionIdFlag string, gcpMetadataClient *metadata.Client) (string, error) {
	// Check if provided via command line flag
	if sessionIdFlag != "" {
		return sessionIdFlag, nil
	}

	// Check if provided via environment variable
	if envSessionId := os.Getenv(types.EnvSessionID); envSessionId != "" {
		return envSessionId, nil
	}

	// Try creating it from GCP metadata
	sessionId, err := gcp.CreateSessionIdentifier(gcpMetadataClient)
	if err == nil {
		return sessionId, nil
	}

	// Fall back to local hostname if GCP metadata fails
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("couldn't determine session identifier: %w", err)
	}

	// Use hostname as fallback
	slog.Info("Using local hostname as session identifier", "hostname", hostname)
	return hostname, nil
}

// GetCredentials retrieves temporary AWS credentials using GCP identity token
func GetCredentials(ctx context.Context, stsRegion, awsAssumeRoleArn, sessionIdentifier string, gcpTokenRetriever gcp.CustomIdentityTokenRetriever) (*types.AWSTempCredentials, error) {
	assumeRoleCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(stsRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsAssumeClient := sts.NewFromConfig(assumeRoleCfg)
	awsCredsCache := aws.NewCredentialsCache(
		stscreds.NewWebIdentityRoleProvider(
			stsAssumeClient,
			awsAssumeRoleArn,
			gcpTokenRetriever,
			func(o *stscreds.WebIdentityRoleOptions) {
				o.RoleSessionName = sessionIdentifier
			},
		),
	)

	awsCredentials, err := awsCredsCache.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &types.AWSTempCredentials{
		Version:         1,
		AccessKeyId:     awsCredentials.AccessKeyID,
		SecretAccessKey: awsCredentials.SecretAccessKey,
		SessionToken:    awsCredentials.SessionToken,
		Expiration:      awsCredentials.Expires,
	}, nil
}
