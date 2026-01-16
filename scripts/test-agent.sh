#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SOCKET=${GLASSHOUSE_AGENT_SOCKET:-/tmp/glasshouse-agent.sock}
RECEIPTS_DIR=${GLASSHOUSE_RECEIPTS_DIR:-/tmp/glasshouse-receipts}
START_AGENT=${GLASSHOUSE_START_AGENT:-0}
AGENT_BIN=${GLASSHOUSE_AGENT_BIN:-${ROOT_DIR}/glasshouse-agent}

if [[ "${START_AGENT}" == "1" ]]; then
  if [[ ! -x "${AGENT_BIN}" ]]; then
    echo "glasshouse-agent binary not found; building" >&2
    (cd "${ROOT_DIR}" && go build -o "${AGENT_BIN}" ./cmd/glasshouse-agent)
  fi
  echo "starting glasshouse-agent on ${SOCKET}" >&2
  sudo "${AGENT_BIN}" start --control-socket "${SOCKET}" --receipt-dir "${RECEIPTS_DIR}" &
  AGENT_PID=$!
  trap 'sudo kill ${AGENT_PID} >/dev/null 2>&1 || true' EXIT
  sleep 1
fi

if [[ ! -S "${SOCKET}" ]]; then
  echo "control socket not found at ${SOCKET}; start the agent or set GLASSHOUSE_AGENT_SOCKET" >&2
  exit 1
fi

if [[ "$#" -gt 0 ]]; then
  CMD=("$@")
else
  CMD=(bash -c "sleep 2; ls /; cat /etc/hosts")
fi

"${CMD[@]}" &
PID=$!
CGPATH=$(awk -F: '$1=="0"{print $3}' "/proc/${PID}/cgroup" 2>/dev/null || true)
CGID=0
if [[ -n "${CGPATH}" && -e "/sys/fs/cgroup${CGPATH}" ]]; then
  CGID=$(stat -c '%i' "/sys/fs/cgroup${CGPATH}")
fi
STARTED_AT=$(date -Iseconds)

PID=${PID} CGID=${CGID} STARTED_AT=${STARTED_AT} python3 - <<'PY'
import json, socket, os
msg = {
    "action": "start",
    "root_pid": int(os.environ["PID"]),
    "cgroup_id": int(os.environ["CGID"]),
    "command": "test-agent",
    "started_at": os.environ["STARTED_AT"],
}
s = socket.socket(socket.AF_UNIX)
s.connect(os.environ.get("GLASSHOUSE_AGENT_SOCKET", "/tmp/glasshouse-agent.sock"))
s.sendall((json.dumps(msg) + "\n").encode())
print(s.recv(4096).decode().strip())
s.close()
PY

wait ${PID}
EXIT_CODE=$?
ENDED_AT=$(date -Iseconds)

PID=${PID} CGID=${CGID} EXIT_CODE=${EXIT_CODE} ENDED_AT=${ENDED_AT} python3 - <<'PY'
import json, socket, os
msg = {
    "action": "end",
    "root_pid": int(os.environ["PID"]),
    "cgroup_id": int(os.environ["CGID"]),
    "exit_code": int(os.environ["EXIT_CODE"]),
    "ended_at": os.environ["ENDED_AT"],
}
s = socket.socket(socket.AF_UNIX)
s.connect(os.environ.get("GLASSHOUSE_AGENT_SOCKET", "/tmp/glasshouse-agent.sock"))
s.sendall((json.dumps(msg) + "\n").encode())
print(s.recv(4096).decode().strip())
s.close()
PY

if [[ "${START_AGENT}" == "1" ]]; then
  if [[ -d "${RECEIPTS_DIR}" ]]; then
    echo "receipts:" >&2
    ls -1 "${RECEIPTS_DIR}" >&2
  fi
else
  echo "note: receipts are written where the running agent is configured" >&2
fi
