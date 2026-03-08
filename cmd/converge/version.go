package main

import (
	"fmt"
	"runtime"

	"github.com/TsekNet/converge/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("converge %s\n", version.Version)
		fmt.Printf("  commit: %s\n", version.Commit)
		fmt.Printf("  built:  %s\n", version.Date)
		fmt.Printf("  go:     %s\n", runtime.Version())
		fmt.Printf("  os:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
