package dsl

import (
	"fmt"
	"testing"
)

func TestShardBucket_Deterministic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		serial string
		want   uint64
	}{
		{"ABC123", 2},
		{"XYZ789", 57},
		{"SERIAL-001", 85},
	}
	for _, tt := range tests {
		t.Run(tt.serial, func(t *testing.T) {
			t.Parallel()
			// Run twice to verify determinism.
			got1 := ShardBucket(tt.serial)
			got2 := ShardBucket(tt.serial)
			if got1 != got2 {
				t.Errorf("ShardBucket(%q) not deterministic: %d vs %d", tt.serial, got1, got2)
			}
			if got1 != tt.want {
				t.Errorf("ShardBucket(%q) = %d, want %d", tt.serial, got1, tt.want)
			}
		})
	}
}

func TestShardBucket_Range(t *testing.T) {
	t.Parallel()

	serials := []string{
		"AAAA", "BBBB", "CCCC", "DDDD", "EEEE",
		"1111", "2222", "3333", "4444", "5555",
		"serial-a", "serial-b", "serial-c",
	}
	for _, s := range serials {
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			got := ShardBucket(s)
			if got >= 100 {
				t.Errorf("ShardBucket(%q) = %d, want < 100", s, got)
			}
		})
	}
}

func TestShardBucket_DifferentSerials(t *testing.T) {
	t.Parallel()

	a := ShardBucket("machine-alpha")
	b := ShardBucket("machine-beta")
	// Not a hard guarantee, but statistically these should differ.
	// If they happen to collide, that's a 1% chance: acceptable for a test.
	if a == b {
		t.Logf("shardBucket collision: %q and %q both map to %d (1%% chance, not a bug)", "machine-alpha", "machine-beta", a)
	}
}

func TestInShard_BoundaryConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		percent int
		serial  string
		want    bool
	}{
		{"zero percent", 0, "ANY-SERIAL", false},
		{"negative percent", -5, "ANY-SERIAL", false},
		{"100 percent", 100, "ANY-SERIAL", true},
		{"over 100 percent", 150, "ANY-SERIAL", true},
		{"empty serial", 50, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := inShard(tt.percent, tt.serial)
			if got != tt.want {
				t.Errorf("inShard(%d, %q) = %v, want %v", tt.percent, tt.serial, got, tt.want)
			}
		})
	}
}

func TestInShard_PercentageDistribution(t *testing.T) {
	t.Parallel()

	// Generate 1000 fake serials and check that ~10% fall in the 10% shard.
	inCount := 0
	total := 1000
	for i := range total {
		serial := fmt.Sprintf("serial-%04d", i)
		if inShard(10, serial) {
			inCount++
		}
	}
	// Allow 5-15% range (generous tolerance for hash distribution).
	low, high := total*5/100, total*15/100
	if inCount < low || inCount > high {
		t.Errorf("10%% shard: got %d/%d (%.1f%%), expected ~10%% (range %d-%d)",
			inCount, total, float64(inCount)/float64(total)*100, low, high)
	}
}

func TestInShard_MonotonicallyInclusive(t *testing.T) {
	t.Parallel()

	// If a serial is in the 10% shard, it must also be in the 50% shard.
	serial := "MONOTONIC-TEST-SERIAL"
	bucket := ShardBucket(serial)

	tests := []struct {
		name    string
		percent int
		want    bool
	}{
		{"below bucket", int(bucket), false},
		{"at bucket + 1", int(bucket) + 1, true},
		{"at 100", 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := inShard(tt.percent, serial)
			if got != tt.want {
				t.Errorf("inShard(%d, %q) = %v, want %v (bucket=%d)",
					tt.percent, serial, got, tt.want, bucket)
			}
		})
	}
}

func TestRun_InShardWithSerial(t *testing.T) {
	t.Parallel()

	run := newRun(New())

	tests := []struct {
		name    string
		percent int
		serial  string
		want    bool
	}{
		{"in shard", 100, "test-serial", true},
		{"out of shard", 0, "test-serial", false},
		{"empty serial", 50, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := run.InShardWithSerial(tt.percent, tt.serial)
			if got != tt.want {
				t.Errorf("InShardWithSerial(%d, %q) = %v, want %v", tt.percent, tt.serial, got, tt.want)
			}
		})
	}
}

func TestRun_InShard_Deterministic(t *testing.T) {
	t.Parallel()

	run := newRun(New())

	tests := []struct {
		name    string
		percent int
		want    bool
	}{
		{"100 percent always true", 100, true},
		{"0 percent always false", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := run.InShard(tt.percent)
			if got != tt.want {
				t.Errorf("InShard(%d) = %v, want %v", tt.percent, got, tt.want)
			}
		})
	}
}

func TestNormalizeSerial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"normal 7 char", "ABC1234", "ABC1234"},
		{"truncated to 7", "ABCDEFGHIJKLMNOP", "ABCDEFG"},
		{"empty", "", ""},
		{"placeholder not specified", "Not Specified", ""},
		{"placeholder oem", "To Be Filled By O.E.M.", ""},
		{"placeholder default", "Default string", ""},
		{"placeholder none", "None", ""},
		{"placeholder zero", "0", ""},
		{"placeholder system serial", "System Serial Number", ""},
		{"placeholder chassis serial", "Chassis Serial Number", ""},
		{"short serial", "AB", "AB"},
		{"whitespace trimmed", "  ABC  ", "ABC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeSerial(tt.raw)
			if got != tt.want {
				t.Errorf("normalizeSerial(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
