#!/usr/bin/env bash
# ==============================================================================
# push_to_turso.bash — Directly push CTF schema and teams to Turso Cloud DB
#
# Usage:
#   TURSO_AUTH_TOKEN="<your_jwt_token>" ./push_to_turso.bash
#   ./push_to_turso.bash "<your_jwt_token>"
# ==============================================================================

set -eo pipefail

TOKEN="${1:-${TURSO_AUTH_TOKEN:-${CTF_DB_AUTH_TOKEN:-}}}"
DB_URL="libsql://ctf-testifywebdev.aws-ap-south-1.turso.io"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}      IEEE CTF — Direct Turso Cloud Database Seeder   ${NC}"
echo -e "${BLUE}======================================================${NC}"

if [ -z "$TOKEN" ]; then
    echo -e "${YELLOW}[?] Enter your Turso Auth Token (JWT):${NC} "
    read -r -s TOKEN
    echo ""
fi

if [ -z "$TOKEN" ]; then
    echo -e "${RED}[!] Error: Turso Auth Token is required to authenticate with Turso Cloud.${NC}"
    echo -e "    You can generate a token using: ${YELLOW}turso db tokens create ctf-testifywebdev${NC}"
    exit 1
fi

export CTF_DB_URL="$DB_URL"
export CTF_DB_AUTH_TOKEN="$TOKEN"
export TURSO_AUTH_TOKEN="$TOKEN"

# 1. Generate clean import CSV if missing
if [ ! -f "teams_import.csv" ]; then
    echo -e "[*] Generating teams_import.csv from ash.csv..."
    python -c "
import csv, re
def slugify(text):
    s = re.sub(r'[^a-z0-9_-]+', '-', text.strip().lower()).strip('-')
    return s or 'team'
seen = set()
teams = []
with open('ash.csv', mode='r', encoding='utf-8') as f:
    for row in csv.DictReader(f):
        name = row.get('Squad Name', '').strip()
        code = row.get('Squad Code', '').strip()
        if not name: continue
        slug = slugify(name)
        base = slug
        c = 2
        while slug in seen:
            slug = f'{base}-{c}'; c += 1
        seen.add(slug)
        teams.append({'name': name, 'ssh_user': slug, 'password': code or 'InitialPass123'})
with open('teams_import.csv', mode='w', newline='', encoding='utf-8') as f:
    w = csv.DictWriter(f, fieldnames=['name', 'ssh_user', 'password'])
    w.writeheader()
    w.writerows(teams)
"
fi

echo -e "[*] Target Database: ${YELLOW}${DB_URL}${NC}"
echo -e "[*] Connecting and applying database migrations to Turso..."

# 2. Run migrations and import teams
(
    cd submission-portal
    go run ./cmd/admin db migrate --migrations ./migrations
    echo -e "${GREEN}[+] Database schema migrations applied successfully!${NC}"
    
    echo -e "[*] Syncing CTF rounds configuration..."
    go run ./cmd/admin rounds load --file ./configs/rounds.yaml
    echo -e "${GREEN}[+] Rounds synchronized!${NC}"

    echo -e "[*] Importing 44 teams into Turso database..."
    go run ./cmd/admin teams import --csv ../teams_import.csv
    echo -e "${GREEN}[+] All teams successfully imported into Turso!${NC}"
)

echo ""
echo -e "${GREEN}======================================================${NC}"
echo -e "${GREEN}  ✓ Turso Cloud Database successfully populated!      ${NC}"
echo -e "${GREEN}======================================================${NC}"
