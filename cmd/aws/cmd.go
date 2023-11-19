/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package aws

import (
	"os"

	"janus/cmd"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Get AWS credentials from GCP environment",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Example: `argocd-k8s-auth-crosscloud gke-to-eks --rolearn "arn:aws:iam::123456789012:role/my-role" --stsregion "us-east-2"`,
	Run: func(cmd *cobra.Command, args []string) {
		rolearn, _ := cmd.Flags().GetString("rolearn")
		stsregion, _ := cmd.Flags().GetString("stsregion")

		// fmt.Println(rolearn, stsregion)

		getAwsCredentials(rolearn, stsregion)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the awsCmd.
func Execute() {
	err := awsCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cmd.RootCmd.AddCommand(awsCmd)

	awsCmd.Flags().StringP("rolearn", "r", "", "AWS role ARN to assume (required)")
	awsCmd.Flags().StringP("stsregion", "s", "us-east-1", "AWS STS region to which requests are made (optional)")
	awsCmd.MarkFlagRequired("rolearn")
}
