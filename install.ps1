# money installer for Windows
# Usage: powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/thedavidweng/money/main/install.ps1 | iex"

$ErrorActionPreference = "Stop"
$Repo = "thedavidweng/money"
$Binary = "money"

function Step($msg) { Write-Host "==> $msg" }
function Die($msg)  { Write-Error "ERROR: $msg"; exit 1 }

$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Die "32-bit Windows is not supported."
}

$platformLabel = "windows/$arch"

function Resolve-Version {
    $resp = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $resp.tag_name
}

function Install-Money {
    Step "Installing money ($platformLabel)"

    $version = Resolve-Version
    Step "Latest version: $version"

    $assetVersion = $version -replace "^v", ""
    $asset = "${Binary}_${assetVersion}_windows_${arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$version/$asset"

    $installDir = if ($env:MONEY_INSTALL_DIR) { $env:MONEY_INSTALL_DIR } else {
        Join-Path $env:LOCALAPPDATA "money\bin"
    }

    $tmpDir = Join-Path $env:TEMP "money-install-$([guid]::NewGuid().ToString('N').Substring(0,8))"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null

    try {
        Step "Downloading $asset"
        $archivePath = Join-Path $tmpDir $asset
        try {
            Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing
        } catch {
            Die "The latest release does not provide $asset. Publish a Windows archive or install with Go: go install github.com/thedavidweng/money/cmd/money@latest"
        }

        Step "Installing to $installDir"
        Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force
        $exe = Join-Path $tmpDir "${Binary}.exe"
        if (-not (Test-Path $exe)) {
            Die "Could not find $Binary.exe in archive."
        }
        Copy-Item $exe (Join-Path $installDir "${Binary}.exe") -Force

        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$installDir*") {
            [Environment]::SetEnvironmentVariable("Path", "$installDir;$userPath", "User")
            $env:Path = "$installDir;$env:Path"
            Step "Added $installDir to user PATH"
        }

        $versionOutput = & (Join-Path $installDir "${Binary}.exe") version 2>$null
        Step "Installed $versionOutput"
    } finally {
        Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Uninstall-Money {
    $installDir = if ($env:MONEY_INSTALL_DIR) { $env:MONEY_INSTALL_DIR } else {
        Join-Path $env:LOCALAPPDATA "money\bin"
    }
    $exe = Join-Path $installDir "${Binary}.exe"
    if (Test-Path $exe) {
        Step "Removing $exe"
        Remove-Item $exe -Force
    }
    Step "Uninstalled. Local config, secrets, and database remain in $env:USERPROFILE\.money\"
}

if ($args.Count -gt 0 -and $args[0] -eq "uninstall") {
    Uninstall-Money
} else {
    Install-Money
    Write-Host ""
    Step "Run 'money setup' to get started."
    Step "Run 'money --help' to see available commands."
}
