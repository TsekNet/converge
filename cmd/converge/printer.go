package main

import (
	"github.com/TsekNet/converge/internal/output"
)

func makePrinter() output.Printer {
	switch outputFormat {
	case "serial":
		return output.NewSerialPrinter()
	case "json":
		return output.NewJSONPrinter()
	default:
		return output.NewTerminalPrinter()
	}
}
