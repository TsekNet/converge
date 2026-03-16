package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TsekNet/converge/internal/daemon"
	"github.com/TsekNet/converge/internal/exit"
	"github.com/TsekNet/converge/internal/platform"
	"github.com/spf13/cobra"
)

var (
	once             bool
	maxRetries       int
	convergedTimeout time.Duration
)

var serveCmd = &cobra.Command{
	Use:   "serve [blueprint]",
	Short: "Run as a persistent service, re-converging on drift",
	Long:  "Run as a persistent daemon that monitors all resources for state drift and re-converges immediately. Use --once to exit after initial convergence.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !platform.IsRoot() {
			exitWithError(exit.NotRoot, fmt.Errorf("converge serve requires root/administrator privileges"))
		}

		printer := makePrinter()
		printer.Banner(app.Version())
		printer.BlueprintHeader(args[0])

		run, err := app.BuildGraph(args[0])
		if err != nil {
			exitWithError(exit.Error, err)
		}

		opts := daemon.Options{
			Timeout:          timeout,
			Parallel:         parallel,
			Once:             once,
			MaxRetries:       maxRetries,
			ConvergedTimeout: convergedTimeout,
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
	serveCmd.Flags().BoolVar(&once, "once", false, "exit after initial convergence (CI/Packer mode)")
	serveCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "max retries before marking a resource noncompliant")
	serveCmd.Flags().DurationVar(&convergedTimeout, "converged-timeout", 0, "exit after system is stable for this duration (0 = run forever)")
	rootCmd.AddCommand(serveCmd)
}

