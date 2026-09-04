# PowerShell script to build, tag, and push all CTF challenge Docker images to Docker Hub
param (
    [Parameter(Mandatory=$false)]
    [string]$Username = ""
)

$ErrorActionPreference = "Stop"

if (-not $Username) {
    $Username = Read-Host -Prompt "Enter your Docker Hub username / organization"
}

if (-not $Username) {
    Write-Error "Docker Hub username cannot be empty."
    exit 1
}

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host " Building and Pushing CTF Challenge Images" -ForegroundColor Cyan
Write-Host " Target Namespace: $Username" -ForegroundColor Yellow
Write-Host "=========================================" -ForegroundColor Cyan

$challenges = @(
    @{ Name = "pwn-maze-oob"; Dir = "pwn-maze-oob"; Tag = "pwn-maze-oob" },
    @{ Name = "pwn-stonks-fmt"; Dir = "pwn-stonks-fmt"; Tag = "pwn-stonks-fmt" },
    @{ Name = "crypto-sphinx-oracle"; Dir = "crypto-sphinx-oracle"; Tag = "crypto-sphinx-oracle" },
    @{ Name = "web-global-megaphone"; Dir = "web-global-megaphone"; Tag = "web-global-megaphone" },
    @{ Name = "web-secure-portal"; Dir = "web-secure-portal"; Tag = "web-secure-portal" }
)

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

foreach ($c in $challenges) {
    $cPath = Join-Path $scriptDir $c.Dir
    $imageTag = "$Username/$($c.Tag):latest"

    Write-Host "`n>>> [1/3] Building image for $($c.Name)..." -ForegroundColor Green
    docker build -t $imageTag -f (Join-Path $cPath "Dockerfile") $cPath

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build $($c.Name)"
        exit 1
    }

    Write-Host ">>> [2/3] Successfully built: $imageTag" -ForegroundColor Green
    Write-Host ">>> [3/3] Pushing $imageTag to Docker Hub..." -ForegroundColor Magenta
    docker push $imageTag

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to push $imageTag. Please make sure you are logged in using 'docker login'."
        exit 1
    }
}

Write-Host "`n=========================================" -ForegroundColor Green
Write-Host " All 5 CTF images successfully pushed! " -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
