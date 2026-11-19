# Download GitHub Release deploy bundle and run stack-up.bat (Windows).
param(
    [string] $Version = $env:OKTOPUS_VERSION,
    [string] $InstallDir = $(if ($env:OKTOPUS_INSTALL_DIR) { $env:OKTOPUS_INSTALL_DIR } else { '.\oktopus' }),
    [switch] $Force,
    [switch] $FetchOnly,
    [switch] $Fetch,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]] $StackUpArgs
)

$ErrorActionPreference = 'Stop'
$Repo = if ($env:OKTOPUS_GITHUB_REPO) { $env:OKTOPUS_GITHUB_REPO } else { 'Camyil-89/oktopus' }

function Show-Usage {
    @'
Usage: install.ps1 [-Version VER] [-InstallDir PATH] [-Force] [-FetchOnly] [-Fetch] [-- stack-up args]

  Downloads oktopus-<version>-deploy.tar.gz from GitHub Releases, extracts it,
  then runs scripts\stack-up.bat.

  stack-up: -WriteEnvOnly maps to --write-env-only; -Build maps to --build

Environment: OKTOPUS_VERSION, OKTOPUS_INSTALL_DIR, OKTOPUS_GITHUB_REPO

One-liner (PowerShell):
  irm https://raw.githubusercontent.com/Camyil-89/oktopus/main/scripts/install.ps1 | iex

Non-interactive:
  $env:OKTOPUS_API_HOST='203.0.113.10'; $env:OKTOPUS_API_PORT='8000'; .\scripts\install.ps1
'@
}

if ($StackUpArgs -contains '-h' -or $StackUpArgs -contains '--help' -or $StackUpArgs -contains '/?') {
    Show-Usage
    exit 0
}

$scriptDir = $PSScriptRoot
if ($scriptDir -and (Test-Path -LiteralPath (Join-Path $scriptDir '..\docker-compose.yml'))) {
    $repoRoot = (Resolve-Path (Join-Path $scriptDir '..')).Path
    if (-not $Fetch -and -not $Version -and -not $Force) {
        Write-Host "Using local checkout at $repoRoot (pass -Fetch to download a release bundle instead)."
        Push-Location $repoRoot
        try {
            $bat = Join-Path $repoRoot 'scripts\stack-up.bat'
            if ($StackUpArgs.Count -gt 0) {
                & $bat @StackUpArgs
            } else {
                & $bat
            }
            exit $LASTEXITCODE
        } finally {
            Pop-Location
        }
    }
}

function Get-LatestVersion {
    param([string] $Repository)
    $uri = "https://api.github.com/repos/$Repository/releases/latest"
    $release = Invoke-RestMethod -Uri $uri -Headers @{ 'User-Agent' = 'oktopus-install' }
    if (-not $release.tag_name) {
        throw 'Could not read latest release tag from GitHub.'
    }
    return ($release.tag_name -replace '^v', '')
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw 'docker not found in PATH.'
}

if ($Version) {
    $Version = $Version -replace '^v', ''
} else {
    Write-Host 'Resolving latest release...'
    $Version = Get-LatestVersion -Repository $Repo
}

$installPath = $InstallDir
if (-not [System.IO.Path]::IsPathRooted($installPath)) {
    $installPath = Join-Path (Get-Location).Path $InstallDir
}
if (-not (Test-Path -LiteralPath $installPath)) {
    New-Item -ItemType Directory -Path $installPath -Force | Out-Null
}
$installPath = (Resolve-Path -LiteralPath $installPath).Path

$marker = Join-Path $installPath '.oktopus-bundle-version'
$bundleName = "oktopus-$Version-deploy.tar.gz"
$url = "https://github.com/$Repo/releases/download/v$Version/$bundleName"

$shouldDownload = $true
$composeFile = Join-Path $installPath 'docker-compose.yml'
if ((Test-Path -LiteralPath $composeFile) -and -not $Force) {
    if ((Test-Path -LiteralPath $marker) -and ((Get-Content -LiteralPath $marker -Raw).Trim() -eq $Version)) {
        $shouldDownload = $false
    } elseif (-not (Test-Path -LiteralPath $marker)) {
        Write-Host 'Install directory already contains compose files; use -Force to replace or set -Version.'
        $shouldDownload = $false
    }
}

if ($shouldDownload) {
    $tmp = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "oktopus-$Version-deploy.tar.gz")
    Write-Host "Downloading $url"
    try {
        Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    } catch {
        throw "Download failed. Check version ($Version) and https://github.com/$Repo/releases"
    }
    Write-Host "Extracting to $installPath"
    tar -xzf $tmp -C $installPath
    Set-Content -LiteralPath $marker -Value $Version -NoNewline
    Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
}

if ($FetchOnly) {
    Write-Host "Bundle ready in $installPath"
    exit 0
}

Push-Location $installPath
try {
    $bat = Join-Path $installPath 'scripts\stack-up.bat'
    if ($StackUpArgs.Count -gt 0) {
        & $bat @StackUpArgs
    } else {
        & $bat
    }
    exit $LASTEXITCODE
} finally {
    Pop-Location
}