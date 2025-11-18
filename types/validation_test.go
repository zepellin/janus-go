package types

import (
	"testing"
)

func TestValidateRoleArn(t *testing.T) {
	tests := []struct {
		name    string
		arn     string
		wantErr bool
	}{
		{
			name:    "valid standard ARN",
			arn:     "arn:aws:iam::123456789012:role/MyRole",
			wantErr: false,
		},
		{
			name:    "valid ARN with path",
			arn:     "arn:aws:iam::123456789012:role/service/MyRole",
			wantErr: false,
		},
		{
			name:    "valid ARN with special characters",
			arn:     "arn:aws:iam::123456789012:role/My-Role_Name.123",
			wantErr: false,
		},
		{
			name:    "valid ARN for AWS China",
			arn:     "arn:aws-cn:iam::123456789012:role/MyRole",
			wantErr: false,
		},
		{
			name:    "valid ARN for AWS GovCloud",
			arn:     "arn:aws-us-gov:iam::123456789012:role/MyRole",
			wantErr: false,
		},
		{
			name:    "empty ARN",
			arn:     "",
			wantErr: true,
		},
		{
			name:    "invalid - missing arn prefix",
			arn:     "aws:iam::123456789012:role/MyRole",
			wantErr: true,
		},
		{
			name:    "invalid - wrong service (not iam)",
			arn:     "arn:aws:s3::123456789012:role/MyRole",
			wantErr: true,
		},
		{
			name:    "invalid - wrong resource type (user instead of role)",
			arn:     "arn:aws:iam::123456789012:user/MyUser",
			wantErr: true,
		},
		{
			name:    "invalid - account ID too short",
			arn:     "arn:aws:iam::12345:role/MyRole",
			wantErr: true,
		},
		{
			name:    "invalid - account ID too long",
			arn:     "arn:aws:iam::1234567890123:role/MyRole",
			wantErr: true,
		},
		{
			name:    "invalid - missing role name",
			arn:     "arn:aws:iam::123456789012:role/",
			wantErr: true,
		},
		{
			name:    "invalid - no role separator",
			arn:     "arn:aws:iam::123456789012:MyRole",
			wantErr: true,
		},
		{
			name:    "invalid - random string",
			arn:     "not-an-arn",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleArn(tt.arn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoleArn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSTSRegion(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		wantErr bool
	}{
		// Valid US regions
		{
			name:    "valid us-east-1",
			region:  "us-east-1",
			wantErr: false,
		},
		{
			name:    "valid us-west-2",
			region:  "us-west-2",
			wantErr: false,
		},
		// Valid with uppercase (should normalize)
		{
			name:    "valid US-EAST-1 uppercase",
			region:  "US-EAST-1",
			wantErr: false,
		},
		// Valid with whitespace (should trim)
		{
			name:    "valid with whitespace",
			region:  "  us-east-1  ",
			wantErr: false,
		},
		// Valid Europe regions
		{
			name:    "valid eu-west-1",
			region:  "eu-west-1",
			wantErr: false,
		},
		{
			name:    "valid eu-central-1",
			region:  "eu-central-1",
			wantErr: false,
		},
		// Valid Asia Pacific regions
		{
			name:    "valid ap-southeast-1",
			region:  "ap-southeast-1",
			wantErr: false,
		},
		{
			name:    "valid ap-northeast-1",
			region:  "ap-northeast-1",
			wantErr: false,
		},
		// Valid GovCloud regions
		{
			name:    "valid us-gov-west-1",
			region:  "us-gov-west-1",
			wantErr: false,
		},
		{
			name:    "valid us-gov-east-1",
			region:  "us-gov-east-1",
			wantErr: false,
		},
		// Valid China regions
		{
			name:    "valid cn-north-1",
			region:  "cn-north-1",
			wantErr: false,
		},
		{
			name:    "valid cn-northwest-1",
			region:  "cn-northwest-1",
			wantErr: false,
		},
		// Valid other regions
		{
			name:    "valid ca-central-1",
			region:  "ca-central-1",
			wantErr: false,
		},
		{
			name:    "valid sa-east-1",
			region:  "sa-east-1",
			wantErr: false,
		},
		{
			name:    "valid af-south-1",
			region:  "af-south-1",
			wantErr: false,
		},
		{
			name:    "valid me-south-1",
			region:  "me-south-1",
			wantErr: false,
		},
		{
			name:    "valid il-central-1",
			region:  "il-central-1",
			wantErr: false,
		},
		// Invalid cases
		{
			name:    "empty region",
			region:  "",
			wantErr: true,
		},
		{
			name:    "invalid region format",
			region:  "invalid-region",
			wantErr: true,
		},
		{
			name:    "non-existent region",
			region:  "us-east-99",
			wantErr: true,
		},
		{
			name:    "typo in region",
			region:  "us-eats-1",
			wantErr: true,
		},
		{
			name:    "random string",
			region:  "not-a-region",
			wantErr: true,
		},
		{
			name:    "old region format",
			region:  "us-standard",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSTSRegion(tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSTSRegion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
