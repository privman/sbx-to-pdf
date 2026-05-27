$ErrorActionPreference = "Stop"

$repo = "privman/sbx-to-pdf"
$binary = "sbx2pdf"

# Detect architecture.
$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or
        (Get-CimInstance Win32_Processor).Architecture -eq 12) {
        "arm64"
    } else {
        "amd64"
    }
} else {
    Write-Error "32-bit Windows is not supported."
    exit 1
}

$asset = "${binary}-windows-${arch}.exe"

# Fetch latest release tag.
Write-Host "Fetching latest release..."
$release = Invoke-RestMethod "https://api.github.com/repos/${repo}/releases/latest"
$tag = $release.tag_name

if (-not $tag) {
    Write-Error "Could not determine latest release."
    exit 1
}

$url = "https://github.com/${repo}/releases/download/${tag}/${asset}"
Write-Host "Downloading ${binary} ${tag} (windows/${arch})..."

# Download to a temp file.
$tmp = Join-Path ([IO.Path]::GetTempPath()) "${binary}.exe"
Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing

# Install to user's local bin directory.
$installDir = Join-Path $env:LOCALAPPDATA "sbx2pdf"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}
$dest = Join-Path $installDir "${binary}.exe"
Move-Item -Force $tmp $dest

# Add to PATH if not already there.
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your PATH (restart your terminal to pick it up)."
}

Write-Host "Installed ${binary} ${tag} to ${dest}"
