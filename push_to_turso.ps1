# ==============================================================================
# push_to_turso.ps1 — Directly push CTF schema, rounds, and teams to Turso Cloud DB
#
# Usage:
#   .\push_to_turso.ps1 [-Token "<your_jwt_token>"]
# ==============================================================================

param(
    [string]$Token = ""
)

$ErrorActionPreference = "Stop"
$DB_URL = "libsql://ctf-testifywebdev.aws-ap-south-1.turso.io"

Write-Host "======================================================" -ForegroundColor Cyan
Write-Host "      IEEE CTF — Direct Turso Cloud Database Seeder   " -ForegroundColor Cyan
Write-Host "======================================================" -ForegroundColor Cyan

if (-not $Token) {
    if ($env:TURSO_AUTH_TOKEN) {
        $Token = $env:TURSO_AUTH_TOKEN
    } elseif ($env:CTF_DB_AUTH_TOKEN) {
        $Token = $env:CTF_DB_AUTH_TOKEN
    } else {
        $Token = Read-Host "Enter your Turso Auth Token (JWT)" -AsSecureString
        $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($Token)
        $Token = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
    }
}

if (-not $Token) {
    Write-Host "[!] Error: Turso Auth Token is required to authenticate with Turso Cloud." -ForegroundColor Red
    Write-Host "    You can generate a token using: turso db tokens create ctf-testifywebdev" -ForegroundColor Yellow
    exit 1
}

$env:CTF_DB_URL = $DB_URL
$env:CTF_DB_AUTH_TOKEN = $Token
$env:TURSO_AUTH_TOKEN = $Token

# 1. Generate clean import CSV if missing
if (-not (Test-Path "teams_import.csv")) {
    Write-Host "[*] Generating teams_import.csv from ash.csv..." -ForegroundColor Yellow
    python -c @"
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
"@
}

Write-Host "[*] Target Database: $DB_URL" -ForegroundColor Yellow
Write-Host "[*] Connecting and applying database migrations to Turso..." -ForegroundColor Yellow

Push-Location "submission-portal"
try {
    go run ./cmd/admin db migrate --migrations ./migrations
    Write-Host "[+] Database schema migrations applied successfully!" -ForegroundColor Green

    Write-Host "[*] Syncing CTF rounds configuration..." -ForegroundColor Yellow
    go run ./cmd/admin rounds load --file ./configs/rounds.yaml
    Write-Host "[+] Rounds synchronized!" -ForegroundColor Green

    Write-Host "[*] Importing 44 teams into Turso database..." -ForegroundColor Yellow
    go run ./cmd/admin teams import --csv ../teams_import.csv
    Write-Host "[+] All teams successfully imported into Turso!" -ForegroundColor Green
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "======================================================" -ForegroundColor Green
Write-Host "  ✓ Turso Cloud Database successfully populated!      " -ForegroundColor Green
Write-Host "======================================================" -ForegroundColor Green
