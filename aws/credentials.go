package aws

import (
	"context"
	"fmt"

	"janus/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"janus/gcp"
	"janus/types"
)

// GetCredentials retrieves temporary AWS credentials using GCP identity token
func GetCredentials(ctx context.Context, stsRegion, awsAssumeRoleArn, sessionIdentifier string, gcpTokenRetriever gcp.CustomIdentityTokenRetriever) (*types.AWSTempCredentials, error) {
	logger.Logger.Debug("Creating AWS STS configuration for region", "StsRegion", stsRegion)
	assumeRoleCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(stsRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsAssumeClient := sts.NewFromConfig(assumeRoleCfg)
	logger.Logger.Debug("Creating AWS STS client", "roleArn", awsAssumeRoleArn, "StsRegion", stsRegion)
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

	logger.Logger.Debug("Retrieving AWS credentials", "sessionIdentifier", sessionIdentifier)
	awsCredentials, err := awsCredsCache.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	logger.Logger.Debug("Successfully retrieved AWS credentials", "roleArn", awsAssumeRoleArn, "StsRegion", stsRegion, "sessionIdentifier", sessionIdentifier)
	return &types.AWSTempCredentials{
		Version:         1,
		AccessKeyId:     awsCredentials.AccessKeyID,
		SecretAccessKey: awsCredentials.SecretAccessKey,
		SessionToken:    awsCredentials.SessionToken,
		Expiration:      awsCredentials.Expires,
	}, nil
}
