package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var GitVersion string
var BuildTime string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of jsl",
	Long:  `All software has versions.`,
	Run: func(cmd *cobra.Command, args []string) {
		if GitVersion == "" {
			GitVersion = "DevBuild"
		}

		if BuildTime == "" {
			BuildTime = "The Future"
		}

		fmt.Printf("Major Version: 2.0\n")
		fmt.Printf("GitVersion:    %s\n", GitVersion)
		fmt.Printf("BuildTime:     %s\n", BuildTime)
	},
}
