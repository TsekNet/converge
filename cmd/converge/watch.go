package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TsekNet/converge/internal/daemon"
	"github.com/spf13/cobra"
)

var (
	once       bool
	maxRetries int
)

var watchCmd = &cobra.Command{
	Use:   "watch [blueprint]",
	Short: "Watch resources for drift and re-converge continuously",
	Long:  "Run as a persistent daemon that watches all resources for state drift and re-converges immediately. Use --once to exit after initial convergence.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		printer := makePrinter()
		printer.Banner(app.Version())
		printer.BlueprintHeader(args[0])

		run, err := app.BuildGraph(args[0])
		if err != nil {
			exitWithError(1, err)
		}

		opts := daemon.Options{
			Timeout:    timeout,
			Parallel:   parallel,
			Once:       once,
			MaxRetries: maxRetries,
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		d := daemon.New(run, printer, opts)
		if err := d.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	watchCmd.Flags().BoolVar(&once, "once", false, "exit after initial convergence (CI/Packer mode)")
	watchCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "max retries before marking a resource noncompliant")
	rootCmd.AddCommand(watchCmd)
}
