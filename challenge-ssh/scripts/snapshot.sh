#!/bin/bash
# challenge-ssh/scripts/snapshot.sh
# Boot the VM once, let it fully initialize, then snapshot for instant restores.
#
# NOTE: this drives the Firecracker API directly over a unix socket — useful as
# an operator escape hatch. The normal workflow is `orchestrator snapshot`.

set -euo pipefail

SOCKET="/tmp/firecracker-golden.sock"
SNAPSHOT_PATH="${SNAPSHOT_PATH:-golden_snapshot}"
MEMFILE_PATH="${MEMFILE_PATH:-golden_mem}"

echo "=== Pausing VM ==="
curl --unix-socket "$SOCKET" -i -X PATCH \
  'http://localhost/vm' \
  -H "Content-Type: application/json" \
  -d '{ "state": "Paused" }'

echo "=== Creating golden snapshot ==="
curl --unix-socket "$SOCKET" -i -X PUT \
  'http://localhost/snapshot/create' \
  -H "Content-Type: application/json" \
  -d '{
    "snapshot_type": "Full",
    "snapshot_path": "./'"$SNAPSHOT_PATH"'",
    "mem_file_path": "./'"$MEMFILE_PATH"'"
  }'

echo "=== Snapshot saved ==="
echo "Restore time: ~1-5ms"
echo "Use (per session): curl --unix-socket <new_socket> -X PUT /snapshot/load"
