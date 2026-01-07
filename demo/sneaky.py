#!/usr/bin/env python3
"""
Demo script that exercises various system operations to generate
a comprehensive execution receipt with filesystem and network activity.
"""

import json
import os
import random
import shutil
import socket
import subprocess
import tempfile
import time

# Filesystem operations - reads
print("Reading system files...")
try:
    with open("/etc/hosts", "r") as f:
        _ = f.read()
    with open("/etc/passwd", "r") as f:
        _ = f.read(100)  # Read partial
    with open("/proc/version", "r") as f:
        _ = f.read()
except Exception:
    pass

# Filesystem operations - writes
print("Writing files...")
with tempfile.NamedTemporaryFile(mode='w', delete=False, prefix='sneaky_', suffix='.txt') as f:
    temp_file = f.name
    f.write("Secret data\n")
    f.write("More content here\n")

# Write to multiple locations
try:
    with open("/tmp/sneaky_output.txt", "w") as f:
        f.write("output data\n")
    with open("/tmp/sneaky_config.json", "w") as f:
        f.write('{"key": "value"}\n')
except Exception:
    pass

# Directory and file churn
print("Making directories and renames...")
root_dir = tempfile.mkdtemp(prefix="sneaky_dir_")
nested_dir = os.path.join(root_dir, "nested")
try:
    os.makedirs(nested_dir, exist_ok=True)
    with open(os.path.join(nested_dir, "alpha.txt"), "w") as f:
        f.write("alpha\n")
    with open(os.path.join(nested_dir, "beta.txt"), "a") as f:
        f.write("beta\n")
    os.rename(
        os.path.join(nested_dir, "alpha.txt"),
        os.path.join(nested_dir, "alpha_renamed.txt"),
    )
    os.chmod(os.path.join(nested_dir, "alpha_renamed.txt"), 0o600)
except Exception:
    pass

# Symlink and stat
print("Making symlinks and stats...")
link_path = os.path.join(root_dir, "link_to_beta")
try:
    os.symlink(os.path.join(nested_dir, "beta.txt"), link_path)
    _ = os.lstat(link_path)
    _ = os.stat(os.path.join(nested_dir, "beta.txt"))
except Exception:
    pass

# JSON read/write
print("Writing JSON...")
try:
    payload = {"ts": time.time(), "rand": random.randint(1, 100)}
    json_path = os.path.join(root_dir, "payload.json")
    with open(json_path, "w") as f:
        json.dump(payload, f)
    with open(json_path, "r") as f:
        _ = json.load(f)
except Exception:
    pass

# Network operations - multiple connections
print("Making network connections...")
targets = [
    ("1.1.1.1", 443),      # Cloudflare DNS (HTTPS)
    ("8.8.8.8", 53),       # Google DNS (DNS)
    ("example.com", 80),   # HTTP
    ("github.com", 443),   # HTTPS
]

for host, port in targets:
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(1)
        s.connect((host, port))
        s.close()
    except Exception:
        pass

# UDP send attempt
try:
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.sendto(b"ping", ("8.8.8.8", 53))
    s.close()
except Exception:
    pass

# IPv6 connection attempt
try:
    s = socket.socket(socket.AF_INET6, socket.SOCK_STREAM)
    s.settimeout(1)
    s.connect(("2606:4700:4700::1111", 443))  # Cloudflare IPv6
    s.close()
except Exception:
    pass

# Process spawning - create child processes
print("Spawning child processes...")
try:
    # Spawn a subprocess
    subprocess.run(["echo", "child process"], timeout=1, capture_output=True)
    # Spawn another with different command
    subprocess.run(["ls", "-la", "/tmp"], timeout=1, capture_output=True)
    # Spawn a shell
    subprocess.run(["/bin/sh", "-c", "echo sh_child > /tmp/sneaky_shell.txt"], timeout=1)
    # Spawn python one-liner
    subprocess.run(
        ["python3", "-c", "open('/tmp/sneaky_py.txt','w').write('py child\\n')"],
        timeout=1,
    )
except Exception:
    pass

# Read process info
print("Reading process info...")
try:
    with open("/proc/self/cmdline", "rb") as f:
        _ = f.read()
    with open("/proc/self/environ", "rb") as f:
        _ = f.read(200)
except Exception:
    pass

# Cleanup
try:
    os.unlink(temp_file)
    shutil.rmtree(root_dir, ignore_errors=True)
except Exception:
    pass

print("Done!")
