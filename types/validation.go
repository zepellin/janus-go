package types

import (
	"fmt"
	"regexp"
	"strings"
)

// AWS IAM role ARN pattern: arn:aws:iam::123456789012:role/RoleName
// Also supports AWS partitions (aws, aws-cn, aws-us-gov)
var arnPattern = regexp.MustCompile(`^arn:(aws|aws-cn|aws-us-gov):iam::\d{12}:role\/[a-zA-Z0-9+=,.@\-_/]+$`)

// validAWSRegions contains all valid AWS regions as of 2024
// Source: https://docs.aws.amazon.com/general/latest/gr/rande.html
var validAWSRegions = map[string]bool{
	// US regions
	"us-east-1": true, // US East (N. Virginia)
	"us-east-2": true, // US East (Ohio)
	"us-west-1": true, // US West (N. California)
	"us-west-2": true, // US West (Oregon)

	// Africa
	"af-south-1": true, // Africa (Cape Town)

	// Asia Pacific
	"ap-east-1":      true, // Asia Pacific (Hong Kong)
	"ap-south-1":     true, // Asia Pacific (Mumbai)
	"ap-south-2":     true, // Asia Pacific (Hyderabad)
	"ap-northeast-1": true, // Asia Pacific (Tokyo)
	"ap-northeast-2": true, // Asia Pacific (Seoul)
	"ap-northeast-3": true, // Asia Pacific (Osaka)
	"ap-southeast-1": true, // Asia Pacific (Singapore)
	"ap-southeast-2": true, // Asia Pacific (Sydney)
	"ap-southeast-3": true, // Asia Pacific (Jakarta)
	"ap-southeast-4": true, // Asia Pacific (Melbourne)

	// Canada
	"ca-central-1": true, // Canada (Central)
	"ca-west-1":    true, // Canada West (Calgary)

	// Europe
	"eu-central-1": true, // Europe (Frankfurt)
	"eu-central-2": true, // Europe (Zurich)
	"eu-west-1":    true, // Europe (Ireland)
	"eu-west-2":    true, // Europe (London)
	"eu-west-3":    true, // Europe (Paris)
	"eu-north-1":   true, // Europe (Stockholm)
	"eu-south-1":   true, // Europe (Milan)
	"eu-south-2":   true, // Europe (Spain)

	// Middle East
	"me-south-1":   true, // Middle East (Bahrain)
	"me-central-1": true, // Middle East (UAE)

	// South America
	"sa-east-1": true, // South America (São Paulo)

	// AWS GovCloud (US)
	"us-gov-east-1": true, // AWS GovCloud (US-East)
	"us-gov-west-1": true, // AWS GovCloud (US-West)

	// China regions
	"cn-north-1":     true, // China (Beijing)
	"cn-northwest-1": true, // China (Ningxia)

	// Israel
	"il-central-1": true, // Israel (Tel Aviv)
}

// ValidateRoleArn validates that the provided string is a valid AWS IAM role ARN
func ValidateRoleArn(arn string) error {
	if arn == "" {
		return fmt.Errorf("role ARN cannot be empty")
	}

	if !arnPattern.MatchString(arn) {
		return fmt.Errorf("invalid AWS role ARN format: %s (expected format: arn:aws:iam::123456789012:role/RoleName)", arn)
	}

	return nil
}

// ValidateSTSRegion validates that the provided string is a valid AWS region
func ValidateSTSRegion(region string) error {
	if region == "" {
		return fmt.Errorf("STS region cannot be empty")
	}

	// Normalize to lowercase for comparison
	normalizedRegion := strings.ToLower(strings.TrimSpace(region))

	if !validAWSRegions[normalizedRegion] {
		return fmt.Errorf("invalid AWS region: %s (see https://docs.aws.amazon.com/general/latest/gr/rande.html for valid regions)", region)
	}

	return nil
}
