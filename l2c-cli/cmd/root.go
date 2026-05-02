package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "l2c",
	Short: "l2c - Lightweight Local to Cloud Tunnel",
	Long:  `l2c is a lightweight tunneling service that uses Cloudflare Workers to expose your local services to the internet.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Root flags if needed
}
