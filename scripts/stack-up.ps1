param(
    [Parameter(Mandatory = $true)] [string] $Root,
    [Parameter(Mandatory = $true)] [string] $Config,
    [Parameter(Mandatory = $true)] [string] $Project,
    [string] $Channel,
    [string] $ImageTag = 'latest',
    [switch] $WriteOnly,
    [switch] $Build
)

$ErrorActionPreference = 'Stop'
$legacyEnv = Join-Path $Root '.env'
if ($WriteOnly) {
    & "$PSScriptRoot\stack-up-env.ps1" -EnvFile $legacyEnv
    Write-Host "Wrote $legacyEnv"
    return
}

$stateDir = Split-Path -Parent $Config
if (-not (Test-Path -LiteralPath $stateDir)) {
    $acl = New-Object System.Security.AccessControl.DirectorySecurity
    $acl.SetAccessRuleProtection($true, $false)
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent().User
    foreach ($identity in @($currentUser, 'S-1-5-18', 'S-1-5-32-544')) {
        $sid = [Security.Principal.SecurityIdentifier]$identity
        $rule = [Security.AccessControl.FileSystemAccessRule]::new(
            $sid, 'FullControl', 'ContainerInherit, ObjectInherit', 'None', 'Allow'
        )
        $acl.AddAccessRule($rule)
    }
    if ($PSVersionTable.PSEdition -eq 'Core') {
        [void][IO.FileSystemAclExtensions]::CreateDirectory($acl, $stateDir)
    } else {
        [void][IO.Directory]::CreateDirectory($stateDir, $acl)
    }
}

if (-not (Test-Path -LiteralPath $Config)) {
    $pending = Join-Path $stateDir ([IO.Path]::GetRandomFileName())
    try {
        if ((Test-Path -LiteralPath $legacyEnv) -and
            (Select-String -LiteralPath $legacyEnv -Pattern '^\s*(?!TAG\s*=)[A-Za-z_]\w*\s*=' -CaseSensitive -Quiet)) {
            Copy-Item -LiteralPath $legacyEnv -Destination $pending
        } else {
            & "$PSScriptRoot\stack-up-env.ps1" -EnvFile $pending
        }
        if (-not (Test-Path -LiteralPath $pending)) { throw 'Could not generate the stack configuration.' }
        [IO.File]::Move($pending, $Config)
    } finally {
        if (Test-Path -LiteralPath $pending) { Remove-Item -LiteralPath $pending -Force }
    }
}
Set-Content -LiteralPath (Join-Path $stateDir 'install-dir') -Value $Root -Encoding UTF8

$composeFile = Join-Path $Root 'docker-compose.desktop.yml'
$compose = @('compose', '-p', $Project, '--env-file', $Config, '-f', $composeFile)
$originalOktopusImage = $env:OKTOPUS_IMAGE
$originalNginxImage = $env:OKTOPUS_NGINX_IMAGE
try {
    if ($Channel -eq 'dev') {
        $env:OKTOPUS_IMAGE = "ghcr.io/camyil-89/oktopus/oktopus/dev:${ImageTag}"
        $env:OKTOPUS_NGINX_IMAGE = "ghcr.io/camyil-89/oktopus/nginx/dev:${ImageTag}"
    }
    if ($Build -or $env:OKTOPUS_COMPOSE_BUILD -eq '1') {
        Invoke-Docker @compose up --build -d --pull never
    } else {
        Invoke-Docker @compose pull oktopus nginx
        Invoke-Docker @compose up -d
    }
} finally {
    $env:OKTOPUS_IMAGE = $originalOktopusImage
    $env:OKTOPUS_NGINX_IMAGE = $originalNginxImage
}

$values = @{}
foreach ($line in Get-Content -LiteralPath $Config) {
    if ($line -match '^([A-Z_]+)=(.*)$') { $values[$Matches[1]] = $Matches[2] }
}
& "$PSScriptRoot\stack-up-banner.ps1"
Write-Host '========== OKTOPUS =========='
Write-Host "UI:       http://$($values.OKTOPUS_API_HOST):$($values.OKTOPUS_API_PORT)/"
Write-Host "Login:    $($values.OKTOPUS_ADMIN_USERNAME)"
Write-Host "Password: $($values.OKTOPUS_ADMIN_PASSWORD)"
Write-Host "Config:   $Config"
