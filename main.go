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
	showVersion := flag.Bool("version", false, "Print version information")
	awsAssumeRoleArn := flag.String("rolearn", "", "AWS role ARN to assume (required)")
	printIdToken := flag.Bool("printidtoken", false, "Print Google identity token when log level is DEBUG")
	stsRegion := flag.String("stsregion", types.STSRegionDefault, "AWS STS region to which requests are made (optional)")
	sessionId := flag.String("sessionid", "", "AWS session identifier (optional) (defaults AWS_SESSION_IDENTIFIER or GCP metadata)")
	logLevel := flag.String("loglevel", "ERROR", "Logging level (DEBUG, INFO, WARN, ERROR)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %s\nCommit: %s\nBuilt: %s\n",
			types.Version,
			types.Commit,
			types.Date)
		os.Exit(0)
	}

	// Initialize global configuration
	types.SetConfig(types.Config{
		PrintIdToken: *printIdToken,
		LogLevel:     *logLevel,
	})

	logger.InitLogger(*logLevel)

	if *awsAssumeRoleArn == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	gcpMetadataClient := gcp.NewMetadataClient(ctx)
	if gcpMetadataClient == nil {
		logger.Logger.Error("Failed to create GCP metadata client: got nil")
		os.Exit(1)
	}

	sessionIdentifier, err := gcp.GetSessionIdentifier(ctx, *sessionId, gcpMetadataClient)
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

	// AWS CLI config credential_process requires JSON output containing credentials
	// and expiration time
	fmt.Printf("%+v\n", credentials)
}
