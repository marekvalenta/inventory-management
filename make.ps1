<#
.SYNOPSIS
    Windows wrapper for Makefile targets via Git Bash.
.EXAMPLE
    .\make.ps1 dev
    .\make.ps1 build
    .\make.ps1 help
#>
param(
    [Parameter(Position = 0)]
    [string]$Target = "help",

    [Parameter(ValueFromRemainingArguments)]
    [string[]]$ExtraArgs
)

$gitBashPaths = @(
    "$env:ProgramFiles\Git\bin\bash.exe",
    "${env:ProgramFiles(x86)}\Git\bin\bash.exe",
    "$env:LocalAppData\Programs\Git\bin\bash.exe"
)

$gitBash = $gitBashPaths | Where-Object { Test-Path $_ } | Select-Object -First 1

if (-not $gitBash) {
    Write-Error @"
Git Bash not found. Install Git for Windows:
  https://git-scm.com/download/win
Then re-run: .\make.ps1 $Target
"@
    exit 1
}

$makeArgs = (@($Target) + $ExtraArgs) -join " "
Write-Host "Running: make $makeArgs" -ForegroundColor Cyan
& $gitBash -c "make $makeArgs"
exit $LASTEXITCODE
