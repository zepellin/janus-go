package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"janus/aws"
	"janus/gcp"
	"janus/logger"
	"janus/types"
)

func main() {
	awsAssumeRoleArn := flag.String("rolearn", "", "AWS role ARN to assume (required)")
	stsRegion := flag.String("stsregion", types.STSRegionDefault, "AWS STS region to which requests are made (optional)")
	sessionId := flag.String("sessionid", "", "AWS session identifier (optional) (defaults AWS_SESSION_IDENTIFIER or GCP metadata)")
	logLevel := flag.String("loglevel", "ERROR", "Logging level (DEBUG, INFO, WARN, ERROR)")

	flag.Parse()

	logger.InitLogger(*logLevel)

	if *awsAssumeRoleArn == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	gcpMetadataClient := gcp.NewMetadataClient()
	if gcpMetadataClient == nil {
		logger.Logger.Error("Failed to create GCP metadata client: got nil")
		os.Exit(1)
	}

	sessionIdentifier, err := gcp.GetSessionIdentifier(*sessionId, gcpMetadataClient)
	if err != nil {
		logger.Logger.Error(fmt.Errorf("failed to get session identifier: %w", err).Error())
		os.Exit(1)
	}

	// Ensure stsRegion isn't nil
	if stsRegion == nil {
		logger.Logger.Error("stsRegion is nil, cannot proceed")
		os.Exit(1)
	}

	gcpMetadataTokenSource, err := gcp.TokenSource(ctx)
	if err != nil {
		logger.Logger.Error(fmt.Errorf("failed to retrieve GCP identity token: %w", err).Error())
		os.Exit(1)
	}

	gcpMetadataToken := gcp.CustomIdentityTokenRetriever{TokenSource: gcpMetadataTokenSource}

	credentials, err := aws.GetCredentials(ctx, *stsRegion, *awsAssumeRoleArn, sessionIdentifier, gcpMetadataToken)
	if err != nil {
		logger.Logger.Error(err.Error())
		os.Exit(1)
	}

	fmt.Printf("%+v\n", credentials)
}
