package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	listBlueprints bool
	listExtensions bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List extensions and blueprints",
	Run: func(cmd *cobra.Command, args []string) {
		showAll := !listBlueprints && !listExtensions

		if showAll || listExtensions {
			exts := app.Extensions()
			fmt.Println("Extensions:")
			for _, e := range exts {
				fmt.Printf("  %-12s %s\n", e.Name, e.Description)
			}
			if showAll {
				fmt.Println()
			}
		}

		if showAll || listBlueprints {
			bps := app.Blueprints()
			if len(bps) == 0 {
				fmt.Println("No blueprints registered.")
				return
			}
			fmt.Println("Blueprints:")
			for _, bp := range bps {
				fmt.Printf("  %-16s %s\n", bp.Name, bp.Description)
			}
		}
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listBlueprints, "blueprints", "b", false, "show only blueprints")
	listCmd.Flags().BoolVarP(&listExtensions, "extensions", "e", false, "show only extensions")
	rootCmd.AddCommand(listCmd)
}
