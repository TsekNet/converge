package output

import (
	"github.com/TsekNet/converge/extensions"
)

// Printer is the interface for all output formatters.
type Printer interface {
	Banner(version string)
	BlueprintHeader(name string)
	ResourceChecking(ext extensions.Extension, current, total int)
	PlanResult(ext extensions.Extension, state *extensions.State)
	ApplyStart(ext extensions.Extension, current, total int)
	ApplyResult(ext extensions.Extension, result *extensions.Result)
	Summary(changed, ok, failed, total int, durationMs int64)
	PlanSummary(pending, ok, total int)
	Error(ext extensions.Extension, err error)
}
