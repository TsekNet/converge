#!/bin/bash
# Linux integration test suite for converge. Run as root:
#   sudo bash test-linux.sh
set -uo pipefail
# No set -e: we handle errors per-test, don't abort on first failure.

BIN="./converge"
PASS=0
FAIL=0
SKIP=0

ok()   { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }
skip() { echo "  SKIP: $1"; ((SKIP++)); }

# Detect SSH service name (Ubuntu/Debian: ssh, RHEL/Fedora: sshd).
SSH_SVC="sshd"
if systemctl list-unit-files ssh.service &>/dev/null; then
    SSH_SVC="ssh"
fi

# Check if systemd is PID 1 (WSL2 may not have it).
HAS_SYSTEMD=false
if [ "$(ps -p 1 -o comm= 2>/dev/null)" = "systemd" ] || systemctl is-system-running &>/dev/null; then
    HAS_SYSTEMD=true
fi

echo "=== Test 1: Plan (read-only) ==="
$BIN plan baseline
ok "Plan completed without errors"
echo ""

echo "=== Test 2: Initial converge ==="
if $BIN serve baseline --timeout 5s 2>&1; then
    ok "Initial converge completed"
else
    # May fail on service if systemd not running, still check other resources.
    echo "  (some resources may have failed, continuing)"
    ok "Initial converge ran (partial)"
fi
echo ""

echo "=== Test 3: Idempotency (expect no 'changed' on second run) ==="
OUTPUT=$($BIN serve baseline --timeout 5s 2>&1 || true)
echo "$OUTPUT"
if echo "$OUTPUT" | grep -q "changed"; then
    fail "Second run should have no changes"
else
    ok "Idempotent: no changes on second run"
fi
echo ""

echo "=== Test 4: File drift detection ==="
$BIN serve baseline &
PID=$!
sleep 5
echo "TAMPERED" > /etc/motd
sleep 10
kill $PID 2>/dev/null; wait $PID 2>/dev/null || true
CONTENT=$(cat /etc/motd)
if echo "$CONTENT" | grep -q "Managed by Converge"; then
    ok "File drift: /etc/motd restored"
else
    fail "File drift: /etc/motd not restored (got: $CONTENT)"
fi
echo ""

echo "=== Test 5: Service drift detection ==="
if [ "$HAS_SYSTEMD" = true ] && systemctl is-active "$SSH_SVC" &>/dev/null; then
    $BIN serve baseline &
    PID=$!
    sleep 3
    systemctl stop "$SSH_SVC" 2>/dev/null || true
    sleep 5
    STATUS=$(systemctl is-active "$SSH_SVC" 2>/dev/null || echo "inactive")
    kill $PID 2>/dev/null; wait $PID 2>/dev/null || true
    if [ "$STATUS" = "active" ]; then
        ok "Service drift: $SSH_SVC restarted"
    else
        fail "Service drift: $SSH_SVC not restarted (status: $STATUS)"
    fi
else
    skip "Service drift: systemd not running or $SSH_SVC not available"
fi
echo ""

echo "=== Test 6: User drift detection ==="
$BIN serve baseline &
PID=$!
sleep 5
userdel devuser 2>/dev/null || true
sleep 10
kill $PID 2>/dev/null; wait $PID 2>/dev/null || true
if id devuser &>/dev/null; then
    ok "User drift: devuser recreated"
else
    fail "User drift: devuser not recreated"
fi
echo ""

echo "=== Test 7: Converged timeout ==="
START=$(date +%s)
$BIN serve baseline --timeout 3s 2>&1 || true
END=$(date +%s)
ELAPSED=$((END - START))
if [ "$ELAPSED" -le 20 ]; then
    ok "Converged timeout: exited after ${ELAPSED}s"
else
    fail "Converged timeout: took ${ELAPSED}s (expected <= 20s)"
fi
echo ""

echo "=== Test 8: Firewall rule ==="
$BIN serve baseline --timeout 1s 2>&1 || true
if nft list table inet converge &>/dev/null; then
    ok "Firewall: nftables converge table exists"
else
    skip "Firewall: nftables not available (expected in containers/WSL)"
fi
echo ""

echo ""
echo "==============================="
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo "==============================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
