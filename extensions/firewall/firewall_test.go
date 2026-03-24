package firewall

import (
	"testing"

	"github.com/TsekNet/converge/extensions"
)

func TestFirewall_ID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{"Allow SSH", "firewall:Allow SSH"},
		{"Block RDP", "firewall:Block RDP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := New(tt.name, 22, "tcp", "inbound", "allow")
			if got := f.ID(); got != tt.want {
				t.Errorf("ID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFirewall_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		port     int
		protocol string
		action   string
		want     string
	}{
		{"Allow SSH", 22, "tcp", "allow", "Firewall Allow SSH (tcp/22 allow)"},
		{"Block RDP", 3389, "tcp", "block", "Firewall Block RDP (tcp/3389 block)"},
		{"Allow DNS", 53, "udp", "allow", "Firewall Allow DNS (udp/53 allow)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := New(tt.name, tt.port, tt.protocol, "inbound", tt.action)
			if got := f.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFirewall_IsCritical(t *testing.T) {
	t.Parallel()

	f := New("test", 22, "tcp", "inbound", "allow")
	if f.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	f.Critical = true
	if !f.IsCritical() {
		t.Error("IsCritical() should be true when set")
	}
}

func TestFirewall_New_Defaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		port      int
		protocol  string
		direction string
		action    string
	}{
		{"Allow SSH", 22, "tcp", "inbound", "allow"},
		{"Block Outbound", 443, "tcp", "outbound", "block"},
		{"Allow DNS", 53, "udp", "inbound", "allow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := New(tt.name, tt.port, tt.protocol, tt.direction, tt.action)
			if f.Name != tt.name {
				t.Errorf("Name = %q, want %q", f.Name, tt.name)
			}
			if f.Port != tt.port {
				t.Errorf("Port = %d, want %d", f.Port, tt.port)
			}
			if f.Protocol != tt.protocol {
				t.Errorf("Protocol = %q, want %q", f.Protocol, tt.protocol)
			}
			if f.Direction != tt.direction {
				t.Errorf("Direction = %q, want %q", f.Direction, tt.direction)
			}
			if f.Action != tt.action {
				t.Errorf("Action = %q, want %q", f.Action, tt.action)
			}
			if f.State != "present" {
				t.Errorf("State = %q, want %q", f.State, "present")
			}
		})
	}
}

func TestFirewall_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fw      Firewall
		wantErr bool
	}{
		{"valid rule", Firewall{Name: "Allow SSH", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, false},
		{"valid with IP", Firewall{Name: "Allow-Net", Port: 443, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present", Source: "10.0.0.1"}, false},
		{"valid with CIDR", Firewall{Name: "Allow-Sub", Port: 443, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present", Source: "10.0.0.0/8"}, false},
		{"valid port 1", Firewall{Name: "Port-1", Port: 1, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, false},
		{"valid port 65535", Firewall{Name: "Port-Max", Port: 65535, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, false},
		{"valid absent state", Firewall{Name: "Absent", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "absent"}, false},
		{"negative port", Firewall{Name: "test", Port: -1, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, true},
		{"empty name", Firewall{Name: "", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, true},
		{"invalid name", Firewall{Name: "rule|inject", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, true},
		{"port zero", Firewall{Name: "test", Port: 0, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, true},
		{"port too high", Firewall{Name: "test", Port: 70000, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present"}, true},
		{"bad protocol", Firewall{Name: "test", Port: 22, Protocol: "icmp", Direction: "inbound", Action: "allow", State: "present"}, true},
		{"bad direction", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "both", Action: "allow", State: "present"}, true},
		{"bad action", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "reject", State: "present"}, true},
		{"bad source", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present", Source: "not-an-ip"}, true},
		{"bad dest", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present", Dest: "not-an-ip"}, true},
		{"bad state", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "unknown"}, true},
		{"ipv6 source rejected", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present", Source: "::1"}, true},
		{"ipv6 cidr rejected", Firewall{Name: "test", Port: 22, Protocol: "tcp", Direction: "inbound", Action: "allow", State: "present", Source: "fd00::/8"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.fw.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFirewall_New_PanicsOnInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fwName    string
		port      int
		protocol  string
		direction string
		action    string
	}{
		{"empty name", "", 22, "tcp", "inbound", "allow"},
		{"bad port", "test", 0, "tcp", "inbound", "allow"},
		{"bad protocol", "test", 22, "icmp", "inbound", "allow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Error("New() should panic on invalid input")
				}
			}()
			New(tt.fwName, tt.port, tt.protocol, tt.direction, tt.action)
		})
	}
}

func TestBoolToState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input bool
		want  string
	}{
		{true, "present"},
		{false, "absent"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := boolToState(tt.input); got != tt.want {
				t.Errorf("boolToState(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResultChanged(t *testing.T) {
	t.Parallel()

	result, err := resultChanged("rule added")
	if err != nil {
		t.Fatalf("resultChanged() error = %v", err)
	}
	if !result.Changed {
		t.Error("Changed = false, want true")
	}
	if result.Status != extensions.StatusChanged {
		t.Errorf("Status = %v, want StatusChanged", result.Status)
	}
	if result.Message != "rule added" {
		t.Errorf("Message = %q, want %q", result.Message, "rule added")
	}
}

func TestValidateAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"valid IPv4", "10.0.0.1", false},
		{"valid CIDR", "10.0.0.0/8", false},
		{"valid CIDR /32", "192.168.1.1/32", false},
		{"IPv6 rejected", "::1", true},
		{"IPv6 CIDR rejected", "fd00::/8", true},
		{"garbage", "not-an-ip", true},
		{"empty string", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAddr(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			}
		})
	}
}

func TestCheckResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		exists      bool
		wantPresent bool
		wantSync    bool
		wantAction  string
	}{
		{"exists and wanted", true, true, true, ""},
		{"exists but unwanted", true, false, false, "remove"},
		{"missing and wanted", false, true, false, "add"},
		{"missing and unwanted", false, false, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state, err := checkResult("test", tt.exists, tt.wantPresent)
			if err != nil {
				t.Fatalf("checkResult() error = %v", err)
			}
			if state.InSync != tt.wantSync {
				t.Errorf("InSync = %v, want %v", state.InSync, tt.wantSync)
			}
			if !tt.wantSync {
				if len(state.Changes) != 1 {
					t.Fatalf("len(Changes) = %d, want 1", len(state.Changes))
				}
				if state.Changes[0].Action != tt.wantAction {
					t.Errorf("Action = %q, want %q", state.Changes[0].Action, tt.wantAction)
				}
			}
		})
	}
}
