package types

import (
	"fmt"
	"regexp"
	"strings"
)

// AWS IAM role ARN pattern: arn:aws:iam::123456789012:role/RoleName
// Also supports AWS partitions (aws, aws-cn, aws-us-gov)
var arnPattern = regexp.MustCompile(`^arn:(aws|aws-cn|aws-us-gov):iam::\d{12}:role\/[a-zA-Z0-9+=,.@\-_/]+$`)

// Matches standard AWS region format: {area}-{sub}-{number}
// Covers commercial, GovCloud (us-gov-*), and China (cn-*) regions.
var regionPattern = regexp.MustCompile(`^(us(-gov)?|af|ap|ca|eu|me|sa|cn|il)-(central|north|south|east|west|northeast|northwest|southeast|southwest)-\d$`)

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

	normalizedRegion := strings.ToLower(strings.TrimSpace(region))

	if !regionPattern.MatchString(normalizedRegion) {
		return fmt.Errorf("invalid AWS region: %s (see https://docs.aws.amazon.com/general/latest/gr/rande.html for valid regions)", region)
	}

	return nil
}
