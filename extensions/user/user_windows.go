//go:build windows

package user

import (
	"context"
	"fmt"
	osuser "os/user"
	"syscall"
	"unsafe"

	"github.com/TsekNet/converge/extensions"
	"golang.org/x/sys/windows"
)

var (
	netapi32                    = windows.NewLazySystemDLL("netapi32.dll")
	procNetUserAdd              = netapi32.NewProc("NetUserAdd")
	procNetUserDel              = netapi32.NewProc("NetUserDel")
	procNetLocalGroupAddMembers = netapi32.NewProc("NetLocalGroupAddMembers")
	procNetApiBufferFree        = netapi32.NewProc("NetApiBufferFree")
)

// USER_INFO_1 matches the Windows USER_INFO_1 structure.
type userInfo1 struct {
	Name     *uint16
	Password *uint16
	PasswordAge uint32
	Priv     uint32
	HomeDir  *uint16
	Comment  *uint16
	Flags    uint32
	ScriptPath *uint16
}

// LOCALGROUP_MEMBERS_INFO_3 uses a domain\user string.
type localGroupMembersInfo3 struct {
	DomainAndName *uint16
}

const (
	userPrivUser   = 1
	ufScript       = 0x0001
	ufNormalAccount = 0x0200
)

// Apply creates or modifies the user via Win32 NetUserAdd / NetLocalGroupAddMembers.
func (u *User) Apply(_ context.Context) (*extensions.Result, error) {
	_, err := lookupUser(u.Name)
	if err != nil {
		return u.createUser()
	}
	return u.ensureGroups()
}

func (u *User) createUser() (*extensions.Result, error) {
	namePtr, _ := syscall.UTF16PtrFromString(u.Name)
	// Empty password: the user must set it later.
	passPtr, _ := syscall.UTF16PtrFromString("")

	info := userInfo1{
		Name:     namePtr,
		Password: passPtr,
		Priv:     userPrivUser,
		Flags:    ufScript | ufNormalAccount,
	}

	var paramErr uint32
	r, _, _ := procNetUserAdd.Call(
		0, // local server
		1, // level 1
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&paramErr)),
	)
	if r != 0 {
		return nil, fmt.Errorf("NetUserAdd(%s): error %d (param %d)", u.Name, r, paramErr)
	}

	if err := u.addToGroups(); err != nil {
		return nil, err
	}

	return &extensions.Result{Changed: true, Status: extensions.StatusChanged, Message: "Created"}, nil
}

func (u *User) ensureGroups() (*extensions.Result, error) {
	if err := u.addToGroups(); err != nil {
		return nil, err
	}
	return &extensions.Result{Changed: false, Status: extensions.StatusOK, Message: "OK"}, nil
}

func (u *User) addToGroups() error {
	for _, group := range u.Groups {
		// Use DOMAIN\user format for local accounts.
		computerName, _ := windows.ComputerName()
		domainUser := computerName + `\` + u.Name
		domainUserPtr, _ := syscall.UTF16PtrFromString(domainUser)

		member := localGroupMembersInfo3{
			DomainAndName: domainUserPtr,
		}

		groupPtr, _ := syscall.UTF16PtrFromString(group)
		r, _, _ := procNetLocalGroupAddMembers.Call(
			0, // local server
			uintptr(unsafe.Pointer(groupPtr)),
			3, // level 3 (LOCALGROUP_MEMBERS_INFO_3)
			uintptr(unsafe.Pointer(&member)),
			1, // total entries
		)
		// ERROR_MEMBER_IN_ALIAS (1378) means already a member, not an error.
		if r != 0 && r != 1378 {
			return fmt.Errorf("NetLocalGroupAddMembers(%s, %s): error %d", group, u.Name, r)
		}
	}
	return nil
}

func shellForUser(_ *osuser.User) string {
	return ""
}
