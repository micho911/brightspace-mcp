#!/usr/bin/env python3
"""Call one tool on a local brightspace-mcp build over stdio.

Usage: scripts/smoke.py [tool [json-arguments]] (default: whoami), e.g.
scripts/smoke.py list_announcements '{"courseId": 123}'. Uses the saved
session, so it talks to the real Brightspace instance.
"""
import json
import os
import subprocess
import sys

tool = sys.argv[1] if len(sys.argv) > 1 else "whoami"
args = json.loads(sys.argv[2]) if len(sys.argv) > 2 else {}
binary = os.path.join(os.getcwd(), "brightspace-mcp.exe") if os.name == "nt" else "./brightspace-mcp"
p = subprocess.Popen([binary], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)


def send(msg):
    p.stdin.write(json.dumps(msg) + "\n")
    p.stdin.flush()


send({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
    "protocolVersion": "2025-06-18", "capabilities": {},
    "clientInfo": {"name": "smoke", "version": "0"}}})
p.stdout.readline()
send({"jsonrpc": "2.0", "method": "notifications/initialized"})
send({"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": tool, "arguments": args}})
result = json.loads(p.stdout.readline())["result"]
p.stdin.close()
p.wait(timeout=10)

print(json.dumps(result.get("structuredContent") or result["content"], indent=2))
sys.exit(1 if result.get("isError") else 0)
