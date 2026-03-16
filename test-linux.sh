#!/bin/bash
# Linux test suite for converge. Run as root:
#   sudo bash test-linux.sh
set -euo pipefail

BIN="./converge"
PASS=0
FAIL=0

ok()   { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }

echo "=== Test 1: Plan (read-only) ==="
$BIN plan baseline
echo ""

echo "=== Test 2: Initial converge ==="
$BIN serve baseline --timeout 5s
echo ""

echo "=== Test 3: Idempotency (run again, expect no changes) ==="
OUTPUT=$($BIN serve baseline --timeout 5s 2>&1)
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
sleep 3
echo "TAMPERED" > /etc/motd
sleep 3
kill $PID 2>/dev/null; wait $PID 2>/dev/null
CONTENT=$(cat /etc/motd)
if [ "$CONTENT" = "Managed by Converge" ]; then
    ok "File drift: /etc/motd restored"
else
    fail "File drift: /etc/motd not restored (got: $CONTENT)"
fi
echo ""

echo "=== Test 5: Service drift detection ==="
$BIN serve baseline &
PID=$!
sleep 3
systemctl stop sshd 2>/dev/null || true
sleep 5
STATUS=$(systemctl is-active sshd 2>/dev/null || echo "inactive")
kill $PID 2>/dev/null; wait $PID 2>/dev/null
if [ "$STATUS" = "active" ]; then
    ok "Service drift: sshd restarted"
else
    fail "Service drift: sshd not restarted (status: $STATUS)"
fi
echo ""

echo "=== Test 6: User drift detection ==="
$BIN serve baseline &
PID=$!
sleep 3
userdel devuser 2>/dev/null || true
sleep 3
kill $PID 2>/dev/null; wait $PID 2>/dev/null
if id devuser &>/dev/null; then
    ok "User drift: devuser recreated"
else
    fail "User drift: devuser not recreated"
fi
echo ""

echo "=== Test 7: Converged timeout ==="
START=$(date +%s)
$BIN serve baseline --timeout 3s
END=$(date +%s)
ELAPSED=$((END - START))
if [ "$ELAPSED" -le 15 ]; then
    ok "Converged timeout: exited after ${ELAPSED}s"
else
    fail "Converged timeout: took ${ELAPSED}s (expected ~3-10s)"
fi
echo ""

echo "=== Test 8: Firewall rule ==="
$BIN serve baseline --timeout 1s
if nft list table inet converge &>/dev/null; then
    ok "Firewall: nftables converge table exists"
else
    fail "Firewall: nftables converge table not found"
fi
echo ""

echo ""
echo "==============================="
echo "Results: $PASS passed, $FAIL failed"
echo "==============================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
