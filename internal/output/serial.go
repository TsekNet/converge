package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
)

type SerialPrinter struct {
	maxNameLen int
	lastGroup  string
}

func NewSerialPrinter() *SerialPrinter {
	return &SerialPrinter{}
}

func (p *SerialPrinter) SetMaxNameLen(n int) {
	p.maxNameLen = n
}

func (p *SerialPrinter) Banner(version string) {
	fmt.Printf(">> converge %s", version)
}

func (p *SerialPrinter) BlueprintHeader(name string) {
	fmt.Printf(" :: %s\n", name)
	fmt.Println("------------------------------------------")
}

func (p *SerialPrinter) ResourceChecking(ext extensions.Extension, current, total int) {
	resType, resName := splitResource(ext.String())
	p.printGroupHeader(resType)
	fmt.Printf("  ... %s [%d/%d]\n", resName, current, total)
}

func (p *SerialPrinter) printGroupHeader(resType string) {
	if resType != p.lastGroup {
		fmt.Printf("  %s\n", resType)
		p.lastGroup = resType
	}
}

func (p *SerialPrinter) PlanResult(ext extensions.Extension, state *extensions.State) {
	_, resName := splitResource(ext.String())

	if state.InSync {
		fmt.Printf("  + %s\n", resName)
		return
	}
	fmt.Printf("  ~ %s\n", resName)
	for _, c := range state.Changes {
		sym := "~"
		if c.Action == "add" {
			sym = "+"
		} else if c.Action == "remove" {
			sym = "-"
		}
		if c.From != "" && c.To != "" {
			fmt.Printf("      %s %s: %s -> %s\n", sym, c.Property, c.From, c.To)
		} else {
			fmt.Printf("      %s %s: %s\n", sym, c.Property, c.To)
		}
	}
}

func (p *SerialPrinter) ApplyStart(_ extensions.Extension, _, _ int) {}

func (p *SerialPrinter) ApplyResult(ext extensions.Extension, result *extensions.Result) {
	resType, resName := splitResource(ext.String())
	p.printGroupHeader(resType)
	dur := formatDuration(result.Duration)
	dots := p.dots(resName)

	if result.Status == extensions.StatusFailed {
		fmt.Printf("  - %s%s%s\n", resName, dots, dur)
		if result.Err != nil {
			fmt.Printf("    Error: %s\n", result.Err.Error())
		}
		return
	}

	if result.Changed && len(result.Changes) > 0 {
		fmt.Printf("  ~ %s%s%s\n", resName, dots, dur)
		for _, c := range result.Changes {
			sym := "~"
			if c.Action == "add" {
				sym = "+"
			} else if c.Action == "remove" {
				sym = "-"
			}
			if c.From != "" && c.To != "" {
				fmt.Printf("      %s %s: %s -> %s\n", sym, c.Property, c.From, c.To)
			} else if c.To != "" {
				fmt.Printf("      %s %s: %s\n", sym, c.Property, c.To)
			}
		}
		return
	}

	fmt.Printf("  + %s%s%s\n", resName, dots, dur)
}

func (p *SerialPrinter) dots(name string) string {
	dotCount := p.maxNameLen + 6 - len(name)
	if dotCount < 3 {
		dotCount = 3
	}
	return " " + strings.Repeat(".", dotCount)
}

func (p *SerialPrinter) Summary(changed, ok, failed, total int, durationMs int64) {
	dur := formatDuration(time.Duration(durationMs) * time.Millisecond)
	fmt.Println("------------------------------------------")

	var parts []string
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d error", failed))
	}
	if changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", changed))
	}
	if ok > 0 {
		parts = append(parts, fmt.Sprintf("%d ok", ok))
	}

	fmt.Printf("  APPLY  %s  (%s)\n", strings.Join(parts, "  "), dur)
}

func (p *SerialPrinter) PlanSummary(pending, ok, total int) {
	fmt.Println("------------------------------------------")

	var parts []string
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d to change", pending))
	}
	if ok > 0 {
		parts = append(parts, fmt.Sprintf("%d ok", ok))
	}

	fmt.Printf("  PLAN  %s\n", strings.Join(parts, "  "))
}

func (p *SerialPrinter) Error(ext extensions.Extension, err error) {
	_, resName := splitResource(ext.String())
	fmt.Printf("  - %s: %s\n", resName, err.Error())
}
