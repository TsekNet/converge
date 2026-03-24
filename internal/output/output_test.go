package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TsekNet/converge/extensions"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestTerminalPrinter_Banner(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewTerminalPrinter()
		p.Banner("0.0.1")
	})
	if out == "" {
		t.Error("Banner() produced no output")
	}
}

func TestTerminalPrinter_ApplyResult(t *testing.T) {
	tests := []struct {
		name   string
		result *extensions.Result
		wantOK bool
	}{
		{"success", &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "updated"}, true},
		{"ok", &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "ok"}, true},
		{"failed", &extensions.Result{Status: extensions.StatusFailed, Err: fmt.Errorf("boom")}, true},
	}

	ext := &stubExt{id: "file:/etc/test", name: "File /etc/test"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				p := NewTerminalPrinter()
				p.SetMaxNameLen(20)
				p.ApplyResult(ext, tt.result)
			})
			if (out != "") != tt.wantOK {
				t.Errorf("ApplyResult() output empty = %v", out == "")
			}
		})
	}
}

func TestSerialPrinter_ApplyResult(t *testing.T) {
	tests := []struct {
		name   string
		result *extensions.Result
	}{
		{"success", &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "updated"}},
		{"ok", &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "ok"}},
		{"failed", &extensions.Result{Status: extensions.StatusFailed, Err: fmt.Errorf("boom")}},
	}

	ext := &stubExt{id: "file:/etc/test", name: "File /etc/test"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				p := NewSerialPrinter()
				p.SetMaxNameLen(20)
				p.ApplyResult(ext, tt.result)
			})
			if out == "" {
				t.Error("ApplyResult() produced no output")
			}
		})
	}
}

func TestSerialPrinter_Banner(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewSerialPrinter()
		p.Banner("0.0.1")
	})
	if out == "" {
		t.Error("Banner() produced no output")
	}
}

func TestSerialPrinter_Summary(t *testing.T) {
	tests := []struct {
		name    string
		changed int
		ok      int
		failed  int
	}{
		{"all ok", 0, 5, 0},
		{"changes", 3, 2, 0},
		{"failures", 1, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				p := NewSerialPrinter()
				p.Summary(tt.changed, tt.ok, tt.failed, tt.changed+tt.ok+tt.failed, 1000)
			})
			if out == "" {
				t.Error("Summary() produced no output")
			}
		})
	}
}

func TestTerminalPrinter_Summary(t *testing.T) {
	tests := []struct {
		name    string
		changed int
		ok      int
		failed  int
	}{
		{"all ok", 0, 5, 0},
		{"changes", 3, 2, 0},
		{"failures", 1, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				p := NewTerminalPrinter()
				p.Summary(tt.changed, tt.ok, tt.failed, tt.changed+tt.ok+tt.failed, 1000)
			})
			if out == "" {
				t.Error("Summary() produced no output")
			}
		})
	}
}

func TestJSONPrinter_Summary(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewJSONPrinter()
		p.BlueprintHeader("test")
		ext := &stubExt{id: "file:/a", name: "File /a"}
		p.ApplyResult(ext, &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "updated"})
		p.Summary(1, 0, 0, 1, 100)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON output is not valid: %v\nOutput: %s", err, out)
	}
	if result["blueprint"] != "test" {
		t.Errorf("blueprint = %v, want 'test'", result["blueprint"])
	}
}

func TestTerminalPrinter_PlanResult(t *testing.T) {
	tests := []struct {
		name  string
		state *extensions.State
	}{
		{"in sync", &extensions.State{InSync: true}},
		{"needs change", &extensions.State{InSync: false, Changes: []extensions.Change{
			{Property: "content", To: "hello", Action: "add"},
		}}},
		{"modify", &extensions.State{InSync: false, Changes: []extensions.Change{
			{Property: "mode", From: "0755", To: "0644", Action: "modify"},
		}}},
		{"remove", &extensions.State{InSync: false, Changes: []extensions.Change{
			{Property: "content", From: "old", Action: "remove"},
		}}},
	}

	ext := &stubExt{id: "file:/etc/test", name: "File /etc/test"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				p := NewTerminalPrinter()
				p.PlanResult(ext, tt.state)
			})
			if out == "" {
				t.Error("PlanResult() produced no output")
			}
		})
	}
}

func TestSerialPrinter_PlanResult(t *testing.T) {
	tests := []struct {
		name  string
		state *extensions.State
	}{
		{"in sync", &extensions.State{InSync: true}},
		{"needs change", &extensions.State{InSync: false, Changes: []extensions.Change{
			{Property: "content", To: "hello", Action: "add"},
			{Property: "mode", From: "0755", To: "0644", Action: "modify"},
		}}},
	}

	ext := &stubExt{id: "file:/etc/test", name: "File /etc/test"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				p := NewSerialPrinter()
				p.PlanResult(ext, tt.state)
			})
			if out == "" {
				t.Error("PlanResult() produced no output")
			}
		})
	}
}

func TestJSONPrinter_PlanResult_Structure(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewJSONPrinter()
		p.BlueprintHeader("test")
		ext1 := &stubExt{id: "file:/etc/a", name: "File /etc/a"}
		ext2 := &stubExt{id: "file:/etc/b", name: "File /etc/b"}
		p.PlanResult(ext1, &extensions.State{InSync: true})
		p.PlanResult(ext2, &extensions.State{
			InSync: false,
			Changes: []extensions.Change{
				{Property: "content", To: "hello", Action: "add"},
				{Property: "mode", From: "0755", To: "0644", Action: "modify"},
			},
		})
		p.PlanSummary(1, 1, 2)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, out)
	}
	resources, ok := result["resources"].([]interface{})
	if !ok || len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %v", result["resources"])
	}

	r0 := resources[0].(map[string]interface{})
	if r0["status"] != "ok" {
		t.Errorf("resources[0].status = %v, want ok", r0["status"])
	}
	if r0["action"] != "in_sync" {
		t.Errorf("resources[0].action = %v, want in_sync", r0["action"])
	}

	r1 := resources[1].(map[string]interface{})
	if r1["status"] != "pending" {
		t.Errorf("resources[1].status = %v, want pending", r1["status"])
	}
	if r1["action"] != "needs_change" {
		t.Errorf("resources[1].action = %v, want needs_change", r1["action"])
	}
	changes, ok := r1["changes"].([]interface{})
	if !ok || len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %v", r1["changes"])
	}
	c0 := changes[0].(map[string]interface{})
	if c0["property"] != "content" || c0["action"] != "add" {
		t.Errorf("changes[0] = %v, want content/add", c0)
	}
	c1 := changes[1].(map[string]interface{})
	if c1["property"] != "mode" || c1["from"] != "0755" || c1["to"] != "0644" {
		t.Errorf("changes[1] = %v, want mode/0755/0644", c1)
	}
}

func TestJSONPrinter_ApplyResult_WithError(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewJSONPrinter()
		p.BlueprintHeader("test")
		ext := &stubExt{id: "file:/etc/a", name: "File /etc/a"}
		p.ApplyResult(ext, &extensions.Result{
			Status:   extensions.StatusFailed,
			Message:  "failed",
			Err:      fmt.Errorf("disk full"),
			Duration: 5 * time.Millisecond,
		})
		p.Summary(0, 0, 1, 1, 100)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, out)
	}
	resources := result["resources"].([]interface{})
	r0 := resources[0].(map[string]interface{})
	errField, ok := r0["error"]
	if !ok || errField == "" {
		t.Errorf("expected error field, got %v", r0)
	}
	if errField != "disk full" {
		t.Errorf("error = %v, want 'disk full'", errField)
	}
}

func TestJSONPrinter_Error_AddsResource(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewJSONPrinter()
		p.BlueprintHeader("test")
		ext := &stubExt{id: "file:/etc/a", name: "File /etc/a"}
		p.Error(ext, fmt.Errorf("check failed"))
		p.Summary(0, 0, 1, 1, 50)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, out)
	}
	resources := result["resources"].([]interface{})
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	r0 := resources[0].(map[string]interface{})
	if r0["status"] != "failed" {
		t.Errorf("status = %v, want failed", r0["status"])
	}
	if r0["error"] != "check failed" {
		t.Errorf("error = %v, want 'check failed'", r0["error"])
	}
}

func TestSerialPrinter_Error(t *testing.T) {
	ext := &stubExt{id: "file:/etc/a", name: "File /etc/a"}
	out := captureStdout(t, func() {
		p := NewSerialPrinter()
		p.Error(ext, fmt.Errorf("permission denied"))
	})
	if !strings.Contains(out, "permission denied") {
		t.Errorf("output %q does not contain error message", out)
	}
}

func TestSerialPrinter_PlanSummary(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewSerialPrinter()
		p.PlanSummary(2, 3, 5)
	})
	if !strings.Contains(out, "PLAN") {
		t.Errorf("output %q does not contain PLAN", out)
	}
	if !strings.Contains(out, "2 to change") {
		t.Errorf("output %q does not contain change count", out)
	}
}

func TestSerialPrinter_ResourceChecking_GroupHeader(t *testing.T) {
	out := captureStdout(t, func() {
		p := NewSerialPrinter()
		ext1 := &stubExt{id: "file:/etc/a", name: "File /etc/a"}
		ext2 := &stubExt{id: "file:/etc/b", name: "File /etc/b"}
		ext3 := &stubExt{id: "pkg:git", name: "Package git"}
		p.ResourceChecking(ext1, 1, 3)
		p.ResourceChecking(ext2, 2, 3)
		p.ResourceChecking(ext3, 3, 3)
	})
	// "File" group header should appear exactly once despite two File resources
	if strings.Count(out, "  File\n") != 1 {
		t.Errorf("expected File group header once, got output:\n%s", out)
	}
	// "Package" group header should appear for the new type
	if !strings.Contains(out, "  Package\n") {
		t.Errorf("expected Package group header, got output:\n%s", out)
	}
}

type stubExt struct {
	id   string
	name string
}

func (s *stubExt) ID() string     { return s.id }
func (s *stubExt) String() string { return s.name }
func (s *stubExt) Check(_ context.Context) (*extensions.State, error) {
	return &extensions.State{InSync: false}, nil
}
func (s *stubExt) Apply(_ context.Context) (*extensions.Result, error) {
	return &extensions.Result{Changed: true}, nil
}

var _ extensions.Extension = (*stubExt)(nil)

// mockExt implements extensions.Extension for testing output
type mockExt struct {
	id   string
	name string
}

func (m *mockExt) ID() string                                          { return m.id }
func (m *mockExt) String() string                                      { return m.name }
func (m *mockExt) Check(_ context.Context) (*extensions.State, error)   { return nil, nil }
func (m *mockExt) Apply(_ context.Context) (*extensions.Result, error) { return nil, nil }

var _ extensions.Extension = (*mockExt)(nil)

func TestTerminalPrinter_AllMethods(t *testing.T) {
	p := NewTerminalPrinter()
	p.SetMaxNameLen(20)
	ext := &mockExt{id: "file:/etc/motd", name: "File /etc/motd"}

	p.Banner("dev")
	p.BlueprintHeader("test")
	p.ResourceChecking(ext, 1, 2)
	// Stop spinner before PlanResult
	p.PlanResult(ext, &extensions.State{InSync: true})
	p.PlanResult(ext, &extensions.State{
		InSync: false,
		Changes: []extensions.Change{
			{Property: "content", From: "old", To: "new", Action: "modify"},
			{Property: "mode", To: "0644", Action: "add"},
		},
	})
	p.ApplyStart(ext, 1, 2)
	p.ApplyResult(ext, &extensions.Result{
		Status: extensions.StatusOK, Message: "ok",
		Duration: 10 * time.Millisecond,
	})
	p.ApplyStart(ext, 2, 2)
	p.ApplyResult(ext, &extensions.Result{
		Status: extensions.StatusFailed, Message: "failed",
		Err: fmt.Errorf("something broke"), Duration: 20 * time.Millisecond,
	})
	p.Summary(1, 1, 1, 3, 100)
	p.Summary(0, 2, 0, 2, 50)
	p.Summary(1, 1, 0, 2, 75)
	p.PlanSummary(0, 2, 2)
	p.PlanSummary(1, 1, 2)
	p.Error(ext, fmt.Errorf("test error"))
}

func TestSerialPrinter_AllMethods(t *testing.T) {
	p := NewSerialPrinter()
	p.SetMaxNameLen(20)
	ext := &mockExt{id: "file:/etc/motd", name: "File /etc/motd"}

	p.Banner("dev")
	p.BlueprintHeader("test")
	p.ResourceChecking(ext, 1, 2)
	p.PlanResult(ext, &extensions.State{InSync: true})
	p.PlanResult(ext, &extensions.State{
		InSync: false,
		Changes: []extensions.Change{
			{Property: "content", From: "old", To: "new", Action: "modify"},
			{Property: "mode", To: "0644", Action: "add"},
			{Property: "owner", From: "root", To: "", Action: "remove"},
		},
	})
	p.ApplyStart(ext, 1, 2)
	p.ApplyResult(ext, &extensions.Result{
		Status: extensions.StatusOK, Message: "ok",
		Duration: 10 * time.Millisecond,
	})
	p.ApplyStart(ext, 2, 2)
	p.ApplyResult(ext, &extensions.Result{
		Status: extensions.StatusFailed, Message: "failed",
		Err: fmt.Errorf("error"), Duration: 20 * time.Millisecond,
	})
	p.Summary(1, 1, 1, 3, 100)
	p.Summary(0, 2, 0, 2, 50)
	p.Summary(1, 1, 0, 2, 75)
	p.PlanSummary(1, 1, 2)
	p.Error(ext, fmt.Errorf("test error"))
}

func TestJSONPrinter_AllMethods(t *testing.T) {
	p := NewJSONPrinter()
	p.SetMaxNameLen(20)
	ext := &mockExt{id: "file:/etc/motd", name: "File /etc/motd"}

	p.Banner("dev")
	p.BlueprintHeader("test")
	p.ResourceChecking(ext, 1, 2)
	p.PlanResult(ext, &extensions.State{InSync: true})
	p.PlanResult(ext, &extensions.State{InSync: false, Changes: []extensions.Change{{Property: "x", To: "y", Action: "add"}}})
	p.ApplyStart(ext, 1, 2)
	p.ApplyResult(ext, &extensions.Result{Status: extensions.StatusOK, Duration: 5 * time.Millisecond})
	p.ApplyResult(ext, &extensions.Result{Status: extensions.StatusFailed, Err: fmt.Errorf("err"), Duration: 5 * time.Millisecond})
	p.ApplyResult(ext, &extensions.Result{Status: extensions.StatusChanged, Duration: 5 * time.Millisecond})
	p.Summary(1, 1, 1, 3, 100)
	p.PlanSummary(1, 1, 2)
	p.Error(ext, fmt.Errorf("test error"))
}

func TestSplitResource(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
		wantName string
	}{
		{"File /etc/motd", "File", "/etc/motd"},
		{"Package git", "Package", "git"},
		{"Service sshd", "Service", "sshd"},
		{"JustAName", "JustAName", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotType, gotName := splitResource(tt.input)
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
		})
	}
}

func TestCapitalizeStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ok", "Ok"},
		{"failed", "Failed"},
		{"", ""},
		{"a", "A"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeStatus(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
