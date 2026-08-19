# Build Ketchup CLI (ketchup + ff aliases)
param(
    [string]$Version = "0.1.0"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)

Push-Location $root
try {
    $ldflags = "-X main.Version=$Version"

    Write-Host "Building ketchup.exe..."
    go build -ldflags $ldflags -o ketchup.exe ./cmd/ketchup

    Write-Host "Building ff.exe (same binary, ff-aware help)..."
    go build -ldflags $ldflags -o ff.exe ./cmd/ketchup

    Write-Host "Done. Binaries:"
    Get-Item ketchup.exe, ff.exe | Format-Table Name, Length
} finally {
    Pop-Location
}
