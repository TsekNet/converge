package main

import (
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [blueprint]",
	Short: "Apply changes to converge the system to desired state",
	Long:  "Run resource checks and apply any needed changes. Requires root/administrator privileges.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		printer := makePrinter()
		printer.Banner(app.Version())
		printer.BlueprintHeader(args[0])

		app.EngineOpts.Timeout = timeout
		app.EngineOpts.Parallel = parallel

		code, err := app.RunApply(args[0], printer)
		if err != nil {
			exitWithError(code, err)
		}
		exitWithCode(code)
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
