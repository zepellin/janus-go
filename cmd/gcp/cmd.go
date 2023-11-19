package gcp

import (
	"os"

	"janus/cmd"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var gcpCmd = &cobra.Command{
	Use:   "gcp",
	Short: "Get Google Cloud credentials from AWS environment",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Example: `janus gcp --gcpserviceaccount "service-account-name@project-name.iam.gserviceaccount.com" --stsregion "us-east-2"`,
	Run: func(cmd *cobra.Command, args []string) {
		worloadidentitypool, _ := cmd.Flags().GetString("worloadidentitypool")
		gcpserviceaccount, _ := cmd.Flags().GetString("gcpserviceaccount")
		stsregion, _ := cmd.Flags().GetString("stsregion")

		getGcpCredentials(worloadidentitypool, gcpserviceaccount, stsregion)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the gcpCmd.
func Execute() {
	err := gcpCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cmd.RootCmd.AddCommand(gcpCmd)

	gcpCmd.Flags().StringP("worloadidentitypool", "p", "", "GCP workload identity pool to use (required)")
	gcpCmd.Flags().StringP("gcpserviceaccount", "a", "", "GCP service account indentity to use (required)")
	gcpCmd.Flags().StringP("stsregion", "s", "us-east-1", "AWS STS region to which requests are made (optional)")
	gcpCmd.MarkFlagRequired("worloadidentitypool")
	gcpCmd.MarkFlagRequired("gcpserviceaccount")
}
