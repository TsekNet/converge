package extensions

import "testing"

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusOK, "ok"},
		{StatusChanged, "changed"},
		{StatusFailed, "failed"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestState(t *testing.T) {
	tests := []struct {
		name       string
		state      State
		wantSync   bool
		wantChanges int
	}{
		{"in sync no changes", State{InSync: true}, true, 0},
		{"in sync with empty changes", State{InSync: true, Changes: []Change{}}, true, 0},
		{"out of sync with changes", State{
			InSync: false,
			Changes: []Change{
				{Property: "content", To: "hello", Action: "add"},
				{Property: "mode", From: "0755", To: "0644", Action: "modify"},
			},
		}, false, 2},
		{"out of sync no details", State{InSync: false}, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", tt.state.InSync, tt.wantSync)
			}
			if len(tt.state.Changes) != tt.wantChanges {
				t.Errorf("len(Changes) = %d, want %d", len(tt.state.Changes), tt.wantChanges)
			}
		})
	}
}
