#!/usr/bin/env bash
# ==============================================================================
# fill.bash — IEEE CTF Team Importer & Database Seeder
# Reads ash.csv (or specified CSV file) and populates teams into the CTF portal.
#
# Usage:
#   ./fill.bash [csv_file]
#
# Environment variables:
#   PASSWORD_MODE  : 'squad_code' (default, uses PT-XXXXX), 'fixed', or 'phone'
#   DEFAULT_PASS   : Password to use when PASSWORD_MODE=fixed (default: 'InitialPass123')
#   EXEC_MODE      : 'docker-compose' (default), 'docker', or 'local'
# ==============================================================================

set -eo pipefail

CSV_FILE="${1:-ash.csv}"
PASSWORD_MODE="${PASSWORD_MODE:-squad_code}"
DEFAULT_PASS="${DEFAULT_PASS:-InitialPass123}"
EXEC_MODE="${EXEC_MODE:-docker-compose}"
CREDENTIALS_OUT="team_credentials.csv"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}        IEEE CTF — Team Database Seeder               ${NC}"
echo -e "${BLUE}======================================================${NC}"

if [ ! -f "$CSV_FILE" ]; then
    echo -e "${RED}[!] CSV file not found: ${CSV_FILE}${NC}"
    exit 1
fi

echo -e "[*] Reading CSV: ${YELLOW}${CSV_FILE}${NC}"
echo -e "[*] Password Mode: ${YELLOW}${PASSWORD_MODE}${NC}"
echo -e "[*] Execution Mode: ${YELLOW}${EXEC_MODE}${NC}"
echo ""

# Ensure service is running if using docker-compose
if [ "$EXEC_MODE" = "docker-compose" ]; then
    if ! docker compose ps --status running 2>/dev/null | grep -q "submission-portal"; then
        echo -e "${YELLOW}[*] Starting submission-portal service via docker compose...${NC}"
        docker compose up -d submission-portal
        sleep 2
    fi
fi

# Find Python interpreter (python3 or python)
if command -v python3 &>/dev/null; then
    PYTHON_BIN="python3"
elif command -v python &>/dev/null; then
    PYTHON_BIN="python"
else
    echo -e "${RED}[!] Python is required to parse CSV reliably. Please install python.${NC}"
    exit 1
fi

export CSV_FILE PASSWORD_MODE DEFAULT_PASS CREDENTIALS_OUT EXEC_MODE

# Generate and process teams using python
$PYTHON_BIN - << 'EOF'
import csv
import os
import re
import subprocess
import sys

csv_file = os.environ.get("CSV_FILE", "ash.csv")
pass_mode = os.environ.get("PASSWORD_MODE", "squad_code")
default_pass = os.environ.get("DEFAULT_PASS", "InitialPass123")
cred_out = os.environ.get("CREDENTIALS_OUT", "team_credentials.csv")
exec_mode = os.environ.get("EXEC_MODE", "docker-compose")

def slugify(text):
    s = text.strip().lower()
    s = re.sub(r'[^a-z0-9_-]+', '-', s)
    s = s.strip('-')
    if not s:
        s = "team"
    return s

teams = []
seen_users = set()

with open(csv_file, mode='r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    for idx, row in enumerate(reader, start=1):
        name = row.get('Squad Name', '').strip()
        code = row.get('Squad Code', '').strip()
        leader_name = row.get('Leader / Contact Name', '').strip()
        leader_email = row.get('Leader Email', '').strip()
        leader_phone = row.get('Leader Phone', '').strip()

        if not name:
            continue

        slug = slugify(name)
        base_slug = slug
        count = 2
        while slug in seen_users:
            slug = f"{base_slug}-{count}"
            count += 1
        seen_users.add(slug)

        if pass_mode == "squad_code" and code:
            password = code
        elif pass_mode == "phone" and leader_phone:
            password = leader_phone
        else:
            password = default_pass

        teams.append({
            "index": idx,
            "name": name,
            "ssh_user": slug,
            "password": password,
            "squad_code": code,
            "leader_name": leader_name,
            "leader_email": leader_email,
            "leader_phone": leader_phone
        })

print(f"[*] Found {len(teams)} teams to import from {csv_file}\n")

# Save credentials CSV
with open(cred_out, mode='w', newline='', encoding='utf-8') as f:
    writer = csv.writer(f)
    writer.writerow(["Index", "Team Name", "SSH User", "Initial Password", "Squad Code", "Leader Name", "Leader Email", "Leader Phone"])
    for t in teams:
        writer.writerow([t["index"], t["name"], t["ssh_user"], t["password"], t["squad_code"], t["leader_name"], t["leader_email"], t["leader_phone"]])

print(f"[*] Saved credentials record in: \033[1;33m{cred_out}\033[0m\n")

success_count = 0
skip_count = 0
fail_count = 0

for t in teams:
    name = t["name"]
    user = t["ssh_user"]
    password = t["password"]

    if exec_mode == "docker-compose":
        cmd = [
            "docker", "compose", "exec", "-T", "submission-portal",
            "./admin", "teams", "create",
            "--name", name,
            "--ssh-user", user,
            "--password", password
        ]
    elif exec_mode == "docker":
        cmd = [
            "docker", "exec", "-i", "submission-portal",
            "./admin", "teams", "create",
            "--name", name,
            "--ssh-user", user,
            "--password", password
        ]
    elif exec_mode == "local":
        if os.path.isfile("./submission-portal/admin"):
            cmd = ["./submission-portal/admin", "teams", "create", "--name", name, "--ssh-user", user, "--password", password]
        elif os.path.isfile("./admin"):
            cmd = ["./admin", "teams", "create", "--name", name, "--ssh-user", user, "--password", password]
        else:
            cmd = ["go", "run", "./submission-portal/cmd/admin", "teams", "create", "--name", name, "--ssh-user", user, "--password", password]

    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, check=False)
        out = (proc.stdout + proc.stderr).strip()
        if proc.returncode == 0:
            print(f"\033[0;32m[+] Created team #{t['index']:02d}: {name:<28} (ssh: {user})\033[0m")
            success_count += 1
        elif "already exists" in out or "UNIQUE constraint failed" in out:
            print(f"\033[1;33m[!] Team #{t['index']:02d} already exists: {name:<28} (ssh: {user})\033[0m")
            skip_count += 1
        else:
            print(f"\033[0;31m[-] Failed team #{t['index']:02d} ({name}): {out}\033[0m")
            fail_count += 1
    except Exception as e:
        print(f"\033[0;31m[-] Error executing for team #{t['index']:02d} ({name}): {e}\033[0m")
        fail_count += 1

print("\n" + "="*50)
print(f"[*] Import complete! Created: {success_count} | Existing/Skipped: {skip_count} | Failed: {fail_count}")
print("="*50)
EOF