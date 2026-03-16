//go:build linux

package condition

import (
	"context"
	"syscall"

	"golang.org/x/sys/unix"
)

// Wait opens a netlink socket subscribed to RTMGRP_LINK and unblocks on
// RTM_NEWLINK / RTM_SETLINK messages, re-checking Met on each.
func (c *networkInterfaceCondition) Wait(ctx context.Context) error {
	if met, _ := c.Met(ctx); met {
		return nil
	}

	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_ROUTE)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	addr := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: unix.RTMGRP_LINK,
	}
	if err := unix.Bind(fd, addr); err != nil {
		return err
	}

	// Set a 500ms receive timeout so we can check ctx cancellation.
	tv := unix.Timeval{Sec: 0, Usec: 500_000}
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		return err
	}

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := unix.Read(fd, buf)
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK || err == unix.ETIMEDOUT {
			continue
		}
		if err != nil {
			return err
		}
		if n < unix.NLMSG_HDRLEN {
			continue
		}

		msgs, err := syscall.ParseNetlinkMessage(buf[:n])
		if err != nil {
			continue
		}
		for _, msg := range msgs {
			if msg.Header.Type == unix.RTM_NEWLINK || msg.Header.Type == unix.RTM_SETLINK {
				if met, _ := c.Met(ctx); met {
					return nil
				}
			}
		}
	}
}
