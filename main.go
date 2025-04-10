package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"janus/aws"
	"janus/gcp"
	"janus/types"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func main() {
	showVersion := flag.Bool("version", false, "Print version information")
	awsAssumeRoleArn := flag.String("rolearn", "", "AWS role ARN to assume (required)")
	stsRegion := flag.String("stsregion", types.STSRegionDefault, "AWS STS region to which requests are made (optional)")
	sessionId := flag.String("sessionid", "", "AWS session identifier (optional) (defaults AWS_SESSION_IDENTIFIER or GCP metadata)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %s\nCommit: %s\nBuilt: %s\n",
			types.Version,
			types.Commit,
			types.Date)
		os.Exit(0)
	}

	if *awsAssumeRoleArn == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	gcpMetadataClient := gcp.NewMetadataClient()
	if gcpMetadataClient == nil {
		logger.Error("Failed to create GCP metadata client: got nil")
		os.Exit(1)
	}

	sessionIdentifier, err := aws.GetSessionIdentifier(*sessionId, gcpMetadataClient)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get session identifier: %w", err).Error())
		os.Exit(1)
	}

	// Ensure stsRegion isn't nil
	if stsRegion == nil {
		logger.Error("stsRegion is nil, cannot proceed")
		os.Exit(1)
	}

	gcpMetadataTokenSource, err := gcp.TokenSource(ctx)
	if err != nil {
		logger.Error(fmt.Errorf("failed to retrieve GCP identity token: %w", err).Error())
		os.Exit(1)
	}

	gcpMetadataToken := gcp.CustomIdentityTokenRetriever{TokenSource: gcpMetadataTokenSource}

	credentials, err := aws.GetCredentials(ctx, *stsRegion, *awsAssumeRoleArn, sessionIdentifier, gcpMetadataToken)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	fmt.Printf("%+v\n", credentials)
}
