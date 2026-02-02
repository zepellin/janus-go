package main

import (
	"context"
	"encoding/json"
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

	logger.InitLogger(*logLevel)

	config := types.Config{
		PrintIdToken: *printIdToken,
		LogLevel:     *logLevel,
	}

	if err := types.ValidateRoleArn(*awsAssumeRoleArn); err != nil {
		logger.Logger.Error(err.Error())
		flag.Usage()
		os.Exit(1)
	}
	if err := types.ValidateSTSRegion(*stsRegion); err != nil {
		logger.Logger.Error(err.Error())
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	gcpMetadataClient := gcp.NewMetadataClient(ctx)

	sessionIdentifier, err := gcp.GetSessionIdentifier(ctx, *sessionId, gcpMetadataClient)
	if err != nil {
		logger.Logger.Error(fmt.Errorf("failed to get session identifier: %w", err).Error())
		os.Exit(1)
	}

	gcpMetadataTokenSource, err := gcp.TokenSource(ctx, config)
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
	if err := json.NewEncoder(os.Stdout).Encode(credentials); err != nil {
		logger.Logger.Error(fmt.Errorf("failed to encode credentials: %w", err).Error())
		os.Exit(1)
	}
}
