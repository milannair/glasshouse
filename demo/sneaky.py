#!/usr/bin/env python3
"""
Demo script that exercises various system operations to generate
a comprehensive execution receipt with filesystem and network activity.
"""

import os
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
except Exception:
    pass

# Cleanup
try:
    os.unlink(temp_file)
except Exception:
    pass

print("Done!")
