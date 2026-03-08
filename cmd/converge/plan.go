package main

import (
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan [blueprint]",
	Short: "Show what would change without making changes",
	Long:  "Run all resource checks and display a diff of pending changes. Does not require root.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		printer := makePrinter()
		printer.Banner(app.Version())
		printer.BlueprintHeader(args[0])

		app.EngineOpts.Timeout = timeout

		code, err := app.RunPlan(args[0], printer)
		if err != nil {
			exitWithError(code, err)
		}
		exitWithCode(code)
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
