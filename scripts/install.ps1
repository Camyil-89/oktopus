# Download GitHub Release deploy bundle and run stack-up.bat (Windows).
param(
    [string] $Version = $env:OKTOPUS_VERSION,
    [string] $InstallDir = $(if ($env:OKTOPUS_INSTALL_DIR) { $env:OKTOPUS_INSTALL_DIR } else { '.\oktopus' }),
    [switch] $Force,
    [switch] $FetchOnly,
    [switch] $Fetch,
    [switch] $LocalStack,
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
  Set TAG=dev in .env to use dev prereleases and images.

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

function Invoke-Docker {
    & docker @args
    if ($LASTEXITCODE -ne 0) { throw "Docker command failed ($LASTEXITCODE)." }
}

function Start-InstallStack ($Root) {
    $runner = Join-Path $Root 'scripts\stack-up.ps1'
    if ($localRunner -and (Test-Path -LiteralPath $localRunner)) {
        $runner = $localRunner
    }
    if (Test-Path -LiteralPath $runner) {
        & $runner -Root $Root -Config $config -Project $project -Channel $channel -ImageTag $imageTag `
            -WriteOnly:($StackUpArgs -contains '--write-env-only' -or $StackUpArgs -contains '-WriteEnvOnly') `
            -Build:($StackUpArgs -contains '--build' -or $StackUpArgs -contains '-Build')
    } else {
        & (Join-Path $Root 'scripts\stack-up.bat') @StackUpArgs
        if ($LASTEXITCODE -ne 0) { throw "Stack startup failed ($LASTEXITCODE)." }
    }
}

$project = 'oktopus-onprem'
$stateDir = Join-Path $env:ProgramData 'Oktopus'
$config = Join-Path $stateDir 'stack.env'
$locationFile = Join-Path $stateDir 'install-dir'
$repoRoot = $null
$localRunner = $null
if ($PSScriptRoot -and (Test-Path -LiteralPath (Join-Path $PSScriptRoot '..\docker-compose.yml'))) {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
    $localRunner = Join-Path $PSScriptRoot 'stack-up.ps1'
}

if (-not $FetchOnly -and $StackUpArgs -notcontains '--write-env-only' -and $StackUpArgs -notcontains '-WriteEnvOnly') {
    $filter = "label=com.docker.compose.project=$project"
    $containers = @(Invoke-Docker ps -aq --filter $filter)
    $installedDir = $null
    if ($containers.Count) {
        $container = (Invoke-Docker inspect $containers[0] | Out-String | ConvertFrom-Json)[0]
        $installedDir = $container.Config.Labels.'com.docker.compose.project.working_dir'
        if (-not $installedDir -or -not [IO.Path]::IsPathRooted($installedDir)) {
            throw 'Cannot determine the Oktopus installation directory.'
        }
    } elseif ((Test-Path -LiteralPath $locationFile) -and @(Invoke-Docker volume ls -q --filter $filter).Count) {
        $installedDir = (Get-Content -LiteralPath $locationFile -Raw).TrimEnd("`r", "`n")
    }

    if ($installedDir) {
        Write-Host "Oktopus installation: $installedDir"
        switch (Read-Host 'Update [u], delete [d], or cancel [Enter]') {
            'u' {
                $InstallDir = $repoRoot = $installedDir
                if (-not $LocalStack -and (Test-Path -LiteralPath (Join-Path $installedDir '.oktopus-bundle-version'))) {
                    $Fetch = $true
                }
            }
            'd' {
                if ((Read-Host 'Type delete to permanently remove Oktopus Docker volumes and their data') -cne 'delete') { exit 0 }
                if ($containers.Count) {
                    $images = @(Invoke-Docker inspect --format '{{.Image}}' @containers | Sort-Object -Unique)
                    Invoke-Docker rm -f @containers
                    foreach ($image in $images) {
                        if (@(Invoke-Docker ps -aq --filter "ancestor=$image").Count) { continue }
                        & docker image rm $image
                        if ($LASTEXITCODE -ne 0) { Write-Warning "Image retained: $image" }
                    }
                }
                $volumes = @(Invoke-Docker volume ls -q --filter $filter)
                if ($volumes.Count) { Invoke-Docker volume rm @volumes }
                $networks = @(Invoke-Docker network ls -q --filter $filter)
                if ($networks.Count) { Invoke-Docker network rm @networks }
                foreach ($file in @($config, $locationFile)) {
                    if (Test-Path -LiteralPath $file) { Remove-Item -LiteralPath $file -Force }
                }
                exit 0
            }
            default { exit 0 }
        }
    }
}

function Read-Channel ($Path) {
    if (Test-Path -LiteralPath $Path -PathType Leaf) {
        foreach ($line in Get-Content -LiteralPath $Path) {
            if ($line -cmatch '^\s*TAG\s*=\s*[''"]?(dev|stable)[''"]?\s*(?:#.*)?$') { $Matches[1] }
        }
    }
}

$channel = Read-Channel (Join-Path (Get-Location).Path '.env') | Select-Object -Last 1
if (-not $channel) { $channel = Read-Channel (Join-Path $InstallDir '.env') | Select-Object -Last 1 }
if (-not $channel -and $repoRoot) { $channel = Read-Channel (Join-Path $repoRoot '.env') | Select-Object -Last 1 }
$marker = Join-Path $InstallDir '.oktopus-bundle-version'
$Version = $Version -replace '^v', ''
if ($channel -eq 'dev' -or $Version -like '*-dev') {
    $channel = 'dev'
    $Fetch = $true
} elseif ((Test-Path -LiteralPath $marker) -and (Get-Content -LiteralPath $marker -Raw).Trim() -like '*-dev') {
    if ($channel -eq 'stable') { $Fetch = $true }
    elseif (-not $channel) { $channel = 'dev' }
}
$imageTag = 'latest'

if ($repoRoot -and -not $FetchOnly -and -not $Fetch -and ($LocalStack -or (-not $Version -and -not $Force))) {
    Start-InstallStack $repoRoot
    exit 0
}

function Get-LatestVersion {
    param([string] $Repository)
    $uri = "https://api.github.com/repos/$Repository/releases/latest"
    if ($channel -eq 'dev') {
        $uri = "https://api.github.com/repos/$Repository/releases"
        for ($page = 1; ; $page++) {
            $releases = Invoke-RestMethod -Uri "${uri}?per_page=100&page=$page" -Headers @{ 'User-Agent' = 'oktopus-install' }
            $latest = $releases | Where-Object { $_.prerelease -and $_.tag_name -like 'v*-dev' } | Select-Object -First 1
            if ($latest) { return ($latest.tag_name -replace '^v', '') }
            if (@($releases).Count -lt 100) { throw 'No dev prerelease found.' }
        }
    }
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
    if ($channel -eq 'dev') {
        $Version = ($Version -replace '-dev$', '') + '-dev'
        $imageTag = $Version
    }
} else {
    $releaseChannel = if ($channel) { $channel } else { 'stable' }
    Write-Host "Resolving latest $releaseChannel release..."
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
    } elseif (-not (Test-Path -LiteralPath $marker) -and $channel -eq 'dev' -and (Test-Path -LiteralPath (Join-Path $installPath '.git'))) {
        throw 'Use -InstallDir with a separate directory, or -Force to replace files in this checkout.'
    } elseif (-not (Test-Path -LiteralPath $marker) -and $channel -ne 'dev') {
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
    if ($LASTEXITCODE -ne 0) { throw "Could not extract the release to $installPath." }
    Set-Content -LiteralPath $marker -Value $Version -NoNewline
    Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
}

if ($FetchOnly) {
    Write-Host "Bundle ready in $installPath"
    exit 0
}

Start-InstallStack $installPath
exit 0
