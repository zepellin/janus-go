package types

import (
	"fmt"
	"regexp"
	"strings"
)

// AWS IAM role ARN pattern: arn:aws:iam::123456789012:role/RoleName
// Also supports AWS partitions (aws, aws-cn, aws-us-gov)
var arnPattern = regexp.MustCompile(`^arn:(aws|aws-cn|aws-us-gov):iam::\d{12}:role\/[a-zA-Z0-9+=,.@\-_/]+$`)

// Matches the shape of an AWS region ({area}-{sub}[-{sub}...]-{number})
// without maintaining a list of known areas, so newly launched regions
// (e.g. mx-central-1) pass validation. Non-existent regions that match the
// shape are rejected by STS itself.
var regionPattern = regexp.MustCompile(`^[a-z]{2,3}(-[a-z]+)+-\d+$`)

// Matches AWS STS RoleSessionName requirements: 2-64 characters of [\w+=,.@-]
var sessionNamePattern = regexp.MustCompile(`^[\w+=,.@-]{2,64}$`)

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

// ValidateSessionIdentifier validates that the provided string satisfies the
// AWS STS RoleSessionName constraints (2-64 characters of [\w+=,.@-])
func ValidateSessionIdentifier(id string) error {
	if !sessionNamePattern.MatchString(id) {
		return fmt.Errorf("invalid session identifier %q (must be 2-64 characters of A-Za-z0-9 and +=,.@_-)", id)
	}

	return nil
}
