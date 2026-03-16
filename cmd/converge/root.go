package main

import (
	"fmt"
	"os"
	"time"

	"github.com/TsekNet/converge/internal/exit"
	"github.com/TsekNet/converge/internal/logging"
	"github.com/TsekNet/converge/internal/version"
	"github.com/spf13/cobra"
)

var outputFormat string
var verbose bool
var timeout time.Duration
var parallel int
var detailedExitCodes bool

var rootCmd = &cobra.Command{
	Use:   version.Name,
	Short: "Desired State Configuration, Compiled",
	Long:  "Converge manages system state using Go blueprints compiled into a single binary.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logging.Init(verbose)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "out", "terminal", "output format: terminal, serial, json")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "detailed output")
	rootCmd.PersistentFlags().DurationVar(&timeout, "resource-timeout", 5*time.Minute, "per-resource timeout for Check/Apply cycles")
	rootCmd.PersistentFlags().IntVar(&parallel, "parallel", 1, "max concurrent resources (1 = sequential)")
	rootCmd.PersistentFlags().BoolVar(&detailedExitCodes, "detailed-exit-codes", false, "use granular exit codes (2=changed, 3=partial, 4=all failed, 5=pending)")
}

func exitWithCode(code int) {
	os.Exit(simplifyExit(code))
}

func exitWithError(code int, err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	os.Exit(simplifyExit(code))
}

func simplifyExit(code int) int {
	if detailedExitCodes {
		return code
	}
	switch code {
	case exit.OK, exit.Changed, exit.Pending:
		return 0
	default:
		return 1
	}
}
