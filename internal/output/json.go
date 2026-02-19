package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/TsekNet/converge/extensions"
)

// JSONPrinter outputs machine-readable JSON for CI/CD pipelines.
type JSONPrinter struct {
	blueprint string
	resources []jsonResource
}

type jsonChange struct {
	Property string `json:"property"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Action   string `json:"action"`
}

type jsonResource struct {
	ID         string       `json:"id"`
	Status     string       `json:"status"`
	Action     string       `json:"action"`
	Changes    []jsonChange `json:"changes,omitempty"`
	DurationMs int64        `json:"duration_ms"`
	Error      string       `json:"error,omitempty"`
}

type jsonOutput struct {
	Blueprint string         `json:"blueprint"`
	Resources []jsonResource `json:"resources"`
	Summary   jsonSummary    `json:"summary"`
}

type jsonSummary struct {
	Changed    int   `json:"changed"`
	Pending    int   `json:"pending,omitempty"`
	OK         int   `json:"ok"`
	Failed     int   `json:"failed"`
	Total      int   `json:"total"`
	DurationMs int64 `json:"duration_ms"`
}

func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{}
}

func (p *JSONPrinter) SetMaxNameLen(_ int) {}

func (p *JSONPrinter) Banner(_ string) {}

func (p *JSONPrinter) BlueprintHeader(name string) {
	p.blueprint = name
}

func (p *JSONPrinter) ResourceChecking(_ extensions.Extension, _, _ int) {}

func (p *JSONPrinter) PlanResult(ext extensions.Extension, state *extensions.State) {
	status := "ok"
	action := "in_sync"
	var changes []jsonChange
	if !state.InSync {
		status = "pending"
		action = "needs_change"
		for _, c := range state.Changes {
			changes = append(changes, jsonChange{
				Property: c.Property,
				From:     c.From,
				To:       c.To,
				Action:   c.Action,
			})
		}
	}
	p.resources = append(p.resources, jsonResource{
		ID:      ext.ID(),
		Status:  status,
		Action:  action,
		Changes: changes,
	})
}

func (p *JSONPrinter) ApplyStart(_ extensions.Extension, _, _ int) {}

func (p *JSONPrinter) ApplyResult(ext extensions.Extension, result *extensions.Result) {
	jr := jsonResource{
		ID:         ext.ID(),
		Status:     result.Status.String(),
		Action:     result.Message,
		DurationMs: result.Duration.Milliseconds(),
	}
	if result.Err != nil {
		jr.Error = result.Err.Error()
	}
	p.resources = append(p.resources, jr)
}

func (p *JSONPrinter) Summary(changed, ok, failed, total int, durationMs int64) {
	out := jsonOutput{
		Blueprint: p.blueprint,
		Resources: p.resources,
		Summary: jsonSummary{
			Changed:    changed,
			OK:         ok,
			Failed:     failed,
			Total:      total,
			DurationMs: durationMs,
		},
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func (p *JSONPrinter) PlanSummary(pending, ok, total int) {
	out := jsonOutput{
		Blueprint: p.blueprint,
		Resources: p.resources,
		Summary: jsonSummary{
			Pending: pending,
			OK:      ok,
			Total:   total,
		},
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func (p *JSONPrinter) Error(ext extensions.Extension, err error) {
	p.resources = append(p.resources, jsonResource{
		ID:     ext.ID(),
		Status: "failed",
		Error:  err.Error(),
	})
}

// Ensure formatDuration is available (defined in terminal.go)
var _ = time.Now
