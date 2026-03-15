package dsl

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"sync"
)

var (
	serialOnce   sync.Once
	cachedSerial string
)

// knownPlaceholders are serial strings commonly returned by VMs and
// bare-metal machines that lack a real serial number.
var knownPlaceholders = map[string]bool{
	"":                        true,
	"0":                       true,
	"not specified":           true,
	"to be filled by o.e.m.": true,
	"default string":         true,
	"none":                   true,
	"system serial number":   true,
	"chassis serial number":  true,
}

// InShard returns true if this machine falls within the given percentage
// (0-100) of the fleet. The shard is computed deterministically from the
// first 7 characters of the hardware serial number using SHA-256, so the
// same machine always lands in the same bucket across runs.
func (r *Run) InShard(percent int) bool {
	return inShard(percent, machineSerial())
}

// InShardWithSerial is like InShard but accepts an explicit serial,
// useful for testing or when the serial is already known. The serial
// is normalized (trimmed, placeholder-checked, truncated to 7 chars)
// identically to InShard, so results are consistent.
func (r *Run) InShardWithSerial(percent int, serial string) bool {
	return inShard(percent, normalizeSerial(serial))
}

// ShardBucket hashes the input and returns a value in [0, 100).
func ShardBucket(serial string) uint64 {
	h := sha256.Sum256([]byte(serial))
	return binary.BigEndian.Uint64(h[:8]) % 100
}

// inShard is the core logic shared by InShard and InShardWithSerial.
func inShard(percent int, serial string) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	if serial == "" {
		return false
	}
	return ShardBucket(serial) < uint64(percent)
}

// machineSerial returns the hardware serial number, cached for the
// lifetime of the process. Returns empty string if detection fails.
// The serial is truncated to 7 characters to normalize across platforms
// (VMs often have 40+ character serials).
func machineSerial() string {
	serialOnce.Do(func() {
		cachedSerial = normalizeSerial(detectSerial())
	})
	return cachedSerial
}

// normalizeSerial trims whitespace, rejects known placeholders, and
// truncates to 7 chars for consistent shard bucketing across platforms.
func normalizeSerial(raw string) string {
	s := strings.TrimSpace(raw)
	if knownPlaceholders[strings.ToLower(s)] {
		return ""
	}
	if len(s) > 7 {
		s = s[:7]
	}
	return s
}
