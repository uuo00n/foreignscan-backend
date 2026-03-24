param(
  [ValidateSet("dev", "prod")]
  [string]$Mode = "dev",
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ExtraDownArgs
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/common.ps1"

try {
  Invoke-DockerDown -Mode $Mode -ExtraDownArgs $ExtraDownArgs
} catch {
  Write-Stderr $_.Exception.Message
  exit 1
}
