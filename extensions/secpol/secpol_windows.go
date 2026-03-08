//go:build windows

package secpol

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
)

var (
	modNetapi32 = windows.NewLazySystemDLL("netapi32.dll")

	procNetUserModalsGet = modNetapi32.NewProc("NetUserModalsGet")
	procNetUserModalsSet = modNetapi32.NewProc("NetUserModalsSet")
	procNetApiBufferFree = modNetapi32.NewProc("NetApiBufferFree")
)

// USER_MODALS_INFO_0 maps to the Win32 struct for password policy (level 0).
type userModalsInfo0 struct {
	MinPasswdLen  uint32
	MaxPasswdAge  uint32
	MinPasswdAge  uint32
	ForceLogoff   uint32
	PasswordHistLen uint32
}

// USER_MODALS_INFO_3 maps to the Win32 struct for account lockout (level 3).
type userModalsInfo3 struct {
	LockoutDuration    uint32
	LockoutObservation uint32
	LockoutThreshold   uint32
}

type SecurityPolicy struct {
	Category string // "password" or "lockout"
	Key      string // field name, e.g. "MinimumPasswordLength"
	Value    string // desired value as string
	Critical bool
}

func New(category, key, value string) *SecurityPolicy {
	return &SecurityPolicy{Category: category, Key: key, Value: value}
}

func (s *SecurityPolicy) ID() string     { return fmt.Sprintf("secpol:%s:%s", s.Category, s.Key) }
func (s *SecurityPolicy) String() string { return fmt.Sprintf("SecurityPolicy %s/%s", s.Category, s.Key) }
func (s *SecurityPolicy) IsCritical() bool { return s.Critical }

func (s *SecurityPolicy) Check(_ context.Context) (*extensions.State, error) {
	current, err := s.readCurrent()
	if err != nil {
		return nil, err
	}

	if current == s.Value {
		return &extensions.State{InSync: true}, nil
	}

	return &extensions.State{
		InSync: false,
		Changes: []extensions.Change{{
			Property: s.Key,
			From:     current,
			To:       s.Value,
			Action:   "modify",
		}},
	}, nil
}

func (s *SecurityPolicy) Apply(_ context.Context) (*extensions.Result, error) {
	if err := s.writeCurrent(); err != nil {
		return nil, err
	}
	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "set"}, nil
}

func (s *SecurityPolicy) readCurrent() (string, error) {
	switch strings.ToLower(s.Category) {
	case "password":
		return s.readPasswordPolicy()
	case "lockout":
		return s.readLockoutPolicy()
	default:
		return "", fmt.Errorf("unsupported secpol category: %q", s.Category)
	}
}

func (s *SecurityPolicy) writeCurrent() error {
	switch strings.ToLower(s.Category) {
	case "password":
		return s.writePasswordPolicy()
	case "lockout":
		return s.writeLockoutPolicy()
	default:
		return fmt.Errorf("unsupported secpol category: %q", s.Category)
	}
}

func (s *SecurityPolicy) readPasswordPolicy() (string, error) {
	var buf *byte
	ret, _, _ := procNetUserModalsGet.Call(0, 0, uintptr(unsafe.Pointer(&buf)))
	if ret != 0 {
		return "", fmt.Errorf("NetUserModalsGet(0): error %d", ret)
	}
	defer procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))

	info := (*userModalsInfo0)(unsafe.Pointer(buf))
	key := strings.ToLower(s.Key)
	switch key {
	case "minimumpasswordlength", "min_passwd_len":
		return strconv.FormatUint(uint64(info.MinPasswdLen), 10), nil
	case "maximumpasswordage", "max_passwd_age":
		return strconv.FormatUint(uint64(info.MaxPasswdAge), 10), nil
	case "minimumpasswordage", "min_passwd_age":
		return strconv.FormatUint(uint64(info.MinPasswdAge), 10), nil
	case "passwordhistorysize", "password_hist_len":
		return strconv.FormatUint(uint64(info.PasswordHistLen), 10), nil
	case "forcelogoff", "force_logoff":
		return strconv.FormatUint(uint64(info.ForceLogoff), 10), nil
	default:
		return "", fmt.Errorf("unknown password policy key: %q", s.Key)
	}
}

func (s *SecurityPolicy) writePasswordPolicy() error {
	var buf *byte
	ret, _, _ := procNetUserModalsGet.Call(0, 0, uintptr(unsafe.Pointer(&buf)))
	if ret != 0 {
		return fmt.Errorf("NetUserModalsGet(0): error %d", ret)
	}
	defer procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))

	info := (*userModalsInfo0)(unsafe.Pointer(buf))
	val, err := strconv.ParseUint(s.Value, 10, 32)
	if err != nil {
		return fmt.Errorf("parse value %q: %w", s.Value, err)
	}
	v := uint32(val)

	key := strings.ToLower(s.Key)
	switch key {
	case "minimumpasswordlength", "min_passwd_len":
		info.MinPasswdLen = v
	case "maximumpasswordage", "max_passwd_age":
		info.MaxPasswdAge = v
	case "minimumpasswordage", "min_passwd_age":
		info.MinPasswdAge = v
	case "passwordhistorysize", "password_hist_len":
		info.PasswordHistLen = v
	case "forcelogoff", "force_logoff":
		info.ForceLogoff = v
	default:
		return fmt.Errorf("unknown password policy key: %q", s.Key)
	}

	var parmErr uint32
	ret, _, _ = procNetUserModalsSet.Call(0, 0, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&parmErr)))
	if ret != 0 {
		return fmt.Errorf("NetUserModalsSet(0): error %d (parm %d)", ret, parmErr)
	}
	return nil
}

func (s *SecurityPolicy) readLockoutPolicy() (string, error) {
	var buf *byte
	ret, _, _ := procNetUserModalsGet.Call(0, 3, uintptr(unsafe.Pointer(&buf)))
	if ret != 0 {
		return "", fmt.Errorf("NetUserModalsGet(3): error %d", ret)
	}
	defer procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))

	info := (*userModalsInfo3)(unsafe.Pointer(buf))
	key := strings.ToLower(s.Key)
	switch key {
	case "lockoutthreshold", "lockout_threshold":
		return strconv.FormatUint(uint64(info.LockoutThreshold), 10), nil
	case "lockoutduration", "lockout_duration":
		return strconv.FormatUint(uint64(info.LockoutDuration), 10), nil
	case "lockoutobservationwindow", "lockout_observation_window":
		return strconv.FormatUint(uint64(info.LockoutObservation), 10), nil
	default:
		return "", fmt.Errorf("unknown lockout policy key: %q", s.Key)
	}
}

func (s *SecurityPolicy) writeLockoutPolicy() error {
	var buf *byte
	ret, _, _ := procNetUserModalsGet.Call(0, 3, uintptr(unsafe.Pointer(&buf)))
	if ret != 0 {
		return fmt.Errorf("NetUserModalsGet(3): error %d", ret)
	}
	defer procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))

	info := (*userModalsInfo3)(unsafe.Pointer(buf))
	val, err := strconv.ParseUint(s.Value, 10, 32)
	if err != nil {
		return fmt.Errorf("parse value %q: %w", s.Value, err)
	}
	v := uint32(val)

	key := strings.ToLower(s.Key)
	switch key {
	case "lockoutthreshold", "lockout_threshold":
		info.LockoutThreshold = v
	case "lockoutduration", "lockout_duration":
		info.LockoutDuration = v
	case "lockoutobservationwindow", "lockout_observation_window":
		info.LockoutObservation = v
	default:
		return fmt.Errorf("unknown lockout policy key: %q", s.Key)
	}

	var parmErr uint32
	ret, _, _ = procNetUserModalsSet.Call(0, 3, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&parmErr)))
	if ret != 0 {
		return fmt.Errorf("NetUserModalsSet(3): error %d (parm %d)", ret, parmErr)
	}
	return nil
}
