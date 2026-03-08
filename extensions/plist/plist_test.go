package plist

import "testing"

func TestPlist_ID(t *testing.T) {
	p := New("com.apple.SoftwareUpdate", "AutomaticCheckEnabled")
	want := "plist:com.apple.SoftwareUpdate:AutomaticCheckEnabled"
	if got := p.ID(); got != want {
		t.Errorf("ID() = %q, want %q", got, want)
	}
}

func TestPlist_String(t *testing.T) {
	p := New("com.apple.SoftwareUpdate", "AutomaticCheckEnabled")
	want := "Plist com.apple.SoftwareUpdate AutomaticCheckEnabled"
	if got := p.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPlist_IsCritical(t *testing.T) {
	p := New("com.apple.SoftwareUpdate", "AutomaticCheckEnabled")
	if p.IsCritical() {
		t.Error("IsCritical() should be false by default")
	}
	p.Critical = true
	if !p.IsCritical() {
		t.Error("IsCritical() should be true after setting")
	}
}

func TestPlist_New_Defaults(t *testing.T) {
	p := New("com.apple.finder", "ShowHardDrivesOnDesktop")
	if p.Domain != "com.apple.finder" {
		t.Errorf("Domain = %q", p.Domain)
	}
	if p.Key != "ShowHardDrivesOnDesktop" {
		t.Errorf("Key = %q", p.Key)
	}
	if p.Host {
		t.Error("Host should default to false")
	}
}
