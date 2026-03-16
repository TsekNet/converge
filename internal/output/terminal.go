package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/TsekNet/converge/extensions"
)

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorGray    = "\033[90m"
	colorBgRed   = "\033[41m"
	colorBgGreen = "\033[42m"
	colorBgCyan  = "\033[46m"
)

type TerminalPrinter struct {
	maxNameLen int
	lastGroup  string
	spinner    *Spinner
}

func NewTerminalPrinter() *TerminalPrinter {
	return &TerminalPrinter{spinner: NewSpinner()}
}

func (p *TerminalPrinter) SetMaxNameLen(n int) {
	p.maxNameLen = n
}

func (p *TerminalPrinter) Banner(version string) {
	fmt.Printf("\n%s»%s %sconverge%s %s%s%s", colorCyan, colorReset, colorBold, colorReset, colorGray, version, colorReset)
}

func (p *TerminalPrinter) BlueprintHeader(name string) {
	fmt.Printf(" %s·%s %s%s%s\n", colorDim, colorReset, colorBold, name, colorReset)
	fmt.Printf("%s────────────────────────────────────────────%s\n", colorDim, colorReset)
}

func (p *TerminalPrinter) ResourceChecking(ext extensions.Extension, current, total int) {
	resType, resName := splitResource(ext.String())
	p.printGroupHeader(resType)
	p.spinner.Start(fmt.Sprintf("%s %s%d/%d%s", resName, colorDim, current, total, colorReset))
}

func (p *TerminalPrinter) printGroupHeader(resType string) {
	if resType != p.lastGroup {
		fmt.Printf("%s%s%s\n", colorBold, resType, colorReset)
		p.lastGroup = resType
	}
}

func (p *TerminalPrinter) PlanResult(ext extensions.Extension, state *extensions.State) {
	p.spinner.Stop()
	_, resName := splitResource(ext.String())

	if state.InSync {
		fmt.Printf("  %s✓%s %s%s%s\n", colorGreen, colorReset, colorDim, resName, colorReset)
		return
	}

	symbol, color := "~", colorYellow
	if len(state.Changes) > 0 && state.Changes[0].Action == "add" {
		symbol, color = "+", colorGreen
	}
	fmt.Printf("  %s%s%s %s%s%s\n", color, symbol, colorReset, colorWhite, resName, colorReset)
	for _, c := range state.Changes {
		sym := "~"
		clr := colorYellow
		if c.Action == "add" {
			sym = "+"
			clr = colorGreen
		} else if c.Action == "remove" {
			sym = "-"
			clr = colorRed
		}
		if c.From != "" && c.To != "" {
			fmt.Printf("      %s%s%s %s: %s%s%s → %s%s%s\n",
				clr, sym, colorReset, c.Property,
				colorRed, c.From, colorReset,
				colorGreen, c.To, colorReset)
		} else if c.To != "" {
			fmt.Printf("      %s%s%s %s: %s%s%s\n",
				clr, sym, colorReset, c.Property, clr, c.To, colorReset)
		}
	}
}

func (p *TerminalPrinter) ApplyStart(ext extensions.Extension, current, total int) {
	resType, resName := splitResource(ext.String())
	p.printGroupHeader(resType)
	p.spinner.Start(fmt.Sprintf("%s %s%d/%d%s", resName, colorDim, current, total, colorReset))
}

func (p *TerminalPrinter) ApplyResult(ext extensions.Extension, result *extensions.Result) {
	p.spinner.Stop()
	_, resName := splitResource(ext.String())
	dur := formatDuration(result.Duration)
	dots := p.dots(resName)

	if result.Status == extensions.StatusFailed {
		fmt.Printf("  %s✗%s %s%s%s%s%s\n",
			colorRed, colorReset, resName, colorDim, dots, dur, colorReset)
		if result.Err != nil {
			fmt.Printf("    %s%s%s\n", colorRed, result.Err.Error(), colorReset)
		}
		return
	}

	if result.Changed && len(result.Changes) > 0 {
		fmt.Printf("  %s~%s %s%s%s%s%s\n",
			colorYellow, colorReset, resName, colorDim, dots, dur, colorReset)
		for _, c := range result.Changes {
			sym, clr := "~", colorYellow
			if c.Action == "add" {
				sym, clr = "+", colorGreen
			} else if c.Action == "remove" {
				sym, clr = "-", colorRed
			}
			if c.From != "" && c.To != "" {
				fmt.Printf("      %s%s%s %s: %s%s%s → %s%s%s\n",
					clr, sym, colorReset, c.Property,
					colorRed, c.From, colorReset,
					colorGreen, c.To, colorReset)
			} else if c.To != "" {
				fmt.Printf("      %s%s%s %s: %s%s%s\n",
					clr, sym, colorReset, c.Property, clr, c.To, colorReset)
			}
		}
		return
	}

	fmt.Printf("  %s✓%s %s%s%s%s%s\n",
		colorGreen, colorReset, resName, colorDim, dots, dur, colorReset)
}

func (p *TerminalPrinter) Summary(changed, ok, failed, total int, durationMs int64) {
	dur := formatDuration(time.Duration(durationMs) * time.Millisecond)
	fmt.Printf("%s────────────────────────────────────────────%s\n", colorDim, colorReset)

	symbol, symbolColor := "✓", colorGreen
	if failed > 0 {
		symbol, symbolColor = "✗", colorRed
	}

	var parts []string
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%s%d error%s", colorRed, failed, colorReset))
	}
	if changed > 0 {
		parts = append(parts, fmt.Sprintf("%s%d changed%s", colorYellow, changed, colorReset))
	}
	if ok > 0 {
		parts = append(parts, fmt.Sprintf("%s%d ok%s", colorGreen, ok, colorReset))
	}

	fmt.Printf("%s%s%s APPLY%s  %s  %s(%s)%s\n",
		colorBold, symbolColor, symbol, colorReset,
		strings.Join(parts, "  "),
		colorDim, dur, colorReset)
	fmt.Println()
}

func (p *TerminalPrinter) PlanSummary(pending, ok, total int) {
	fmt.Printf("%s────────────────────────────────────────────%s\n", colorDim, colorReset)
	if pending == 0 {
		fmt.Printf("%s%s✓ CONVERGED%s  %s%d ok%s\n",
			colorBold, colorGreen, colorReset,
			colorGreen, ok, colorReset)
	} else {
		var parts []string
		if pending > 0 {
			parts = append(parts, fmt.Sprintf("%s%d to change%s", colorYellow, pending, colorReset))
		}
		if ok > 0 {
			parts = append(parts, fmt.Sprintf("%s%d ok%s", colorGreen, ok, colorReset))
		}
		fmt.Printf("%s%s● PLAN%s  %s\n",
			colorBold, colorCyan, colorReset,
			strings.Join(parts, "  "))
		fmt.Printf("%sRun %sconverge serve --timeout 1s%s%s to apply.%s\n",
			colorDim, colorWhite, colorReset, colorDim, colorReset)
	}
	fmt.Println()
}

func (p *TerminalPrinter) Error(ext extensions.Extension, err error) {
	_, resName := splitResource(ext.String())
	fmt.Printf("  %s✗%s %s: %s%s%s\n", colorRed, colorReset, resName, colorRed, err.Error(), colorReset)
}

func (p *TerminalPrinter) dots(name string) string {
	dotCount := p.maxNameLen + 6 - len(name)
	if dotCount < 3 {
		dotCount = 3
	}
	return " " + strings.Repeat(".", dotCount)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
