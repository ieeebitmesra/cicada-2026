# ==============================================================================
# fill.ps1 — IEEE CTF Team Importer & Database Seeder (PowerShell)
# Reads ash.csv (or specified CSV file) and populates teams into the CTF portal.
#
# Usage:
#   .\fill.ps1 [-CsvFile ash.csv] [-PasswordMode squad_code|phone|fixed] [-DefaultPass InitialPass123]
# ==============================================================================

param(
    [string]$CsvFile = "ash.csv",
    [string]$PasswordMode = "squad_code", # 'squad_code', 'phone', or 'fixed'
    [string]$DefaultPass = "InitialPass123",
    [string]$ExecMode = "docker-compose"
)

$ErrorActionPreference = "Stop"

Write-Host "======================================================" -ForegroundColor Cyan
Write-Host "        IEEE CTF — Team Database Seeder               " -ForegroundColor Cyan
Write-Host "======================================================" -ForegroundColor Cyan

if (-not (Test-Path $CsvFile)) {
    Write-Host "[!] CSV file not found: $CsvFile" -ForegroundColor Red
    exit 1
}

Write-Host "[*] Reading CSV: $CsvFile" -ForegroundColor Yellow
Write-Host "[*] Password Mode: $PasswordMode" -ForegroundColor Yellow
Write-Host "[*] Execution Mode: $ExecMode" -ForegroundColor Yellow
Write-Host ""

$env:CSV_FILE = $CsvFile
$env:PASSWORD_MODE = $PasswordMode
$env:DEFAULT_PASS = $DefaultPass
$env:EXEC_MODE = $ExecMode
$env:CREDENTIALS_OUT = "team_credentials.csv"

# Start container if using docker-compose
if ($ExecMode -eq "docker-compose") {
    $running = docker compose ps --status running 2>$null | Select-String "submission-portal"
    if (-not $running) {
        Write-Host "[*] Starting submission-portal service via docker compose..." -ForegroundColor Yellow
        docker compose up -d submission-portal
        Start-Sleep -Seconds 2
    }
}

python -c @"
import csv, os, re, subprocess

csv_file = os.environ.get('CSV_FILE', 'ash.csv')
pass_mode = os.environ.get('PASSWORD_MODE', 'squad_code')
default_pass = os.environ.get('DEFAULT_PASS', 'InitialPass123')
cred_out = os.environ.get('CREDENTIALS_OUT', 'team_credentials.csv')
exec_mode = os.environ.get('EXEC_MODE', 'docker-compose')

def slugify(text):
    s = text.strip().lower()
    s = re.sub(r'[^a-z0-9_-]+', '-', s)
    s = s.strip('-')
    return s or 'team'

teams = []
seen = set()

with open(csv_file, mode='r', encoding='utf-8') as f:
    for idx, row in enumerate(csv.DictReader(f), start=1):
        name = row.get('Squad Name', '').strip()
        code = row.get('Squad Code', '').strip()
        leader_name = row.get('Leader / Contact Name', '').strip()
        leader_email = row.get('Leader Email', '').strip()
        leader_phone = row.get('Leader Phone', '').strip()
        if not name:
            continue
        slug = slugify(name)
        base = slug
        c = 2
        while slug in seen:
            slug = f'{base}-{c}'
            c += 1
        seen.add(slug)
        
        pwd = code if (pass_mode == 'squad_code' and code) else (leader_phone if pass_mode == 'phone' and leader_phone else default_pass)
        teams.append({
            'idx': idx, 'name': name, 'user': slug, 'pass': pwd,
            'code': code, 'lead': leader_name, 'email': leader_email, 'phone': leader_phone
        })

print(f'[*] Found {len(teams)} teams to import from {csv_file}\n')

with open(cred_out, mode='w', newline='', encoding='utf-8') as f:
    w = csv.writer(f)
    w.writerow(['Index', 'Team Name', 'SSH User', 'Initial Password', 'Squad Code', 'Leader Name', 'Leader Email', 'Leader Phone'])
    for t in teams:
        w.writerow([t['idx'], t['name'], t['user'], t['pass'], t['code'], t['lead'], t['email'], t['phone']])

print(f'[*] Saved credentials record in: {cred_out}\n')

created, skipped, failed = 0, 0, 0

for t in teams:
    cmd = ['docker', 'compose', 'exec', '-T', 'submission-portal', './admin', 'teams', 'create', '--name', t['name'], '--ssh-user', t['user'], '--password', t['pass']]
    proc = subprocess.run(cmd, capture_output=True, text=True, check=False)
    out = (proc.stdout + proc.stderr).strip()
    if proc.returncode == 0:
        print(f'[+] Created team #{t[\"idx\"]:02d}: {t[\"name\"]} (ssh: {t[\"user\"]})')
        created += 1
    elif 'already exists' in out or 'UNIQUE constraint' in out:
        print(f'[!] Team #{t[\"idx\"]:02d} already exists: {t[\"name\"]} (ssh: {t[\"user\"]})')
        skipped += 1
    else:
        print(f'[-] Failed team #{t[\"idx\"]:02d} ({t[\"name\"]}): {out}')
        failed += 1

print(f'\n[*] Import complete! Created: {created} | Existing/Skipped: {skipped} | Failed: {failed}')
"@
