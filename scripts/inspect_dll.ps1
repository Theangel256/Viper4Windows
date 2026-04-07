param(
  [string]$InputPath
)

Write-Host "CWD: $(Get-Location)"

# Determine path: prefer explicit InputPath, otherwise look in CWD
if ($InputPath -and (Test-Path $InputPath)) {
  $path = (Resolve-Path $InputPath).Path
} else {
  $path = Join-Path (Get-Location) "Hydrogen_Inst.dll"
}

if (-not (Test-Path $path)) {
  Write-Host "Hydrogen_Inst.dll NOT found at $path"
  exit 2
}

Write-Host "FOUND Hydrogen_Inst.dll at $path"
Get-Item $path | Format-List Name,Length,LastWriteTime
try {
  $hash = Get-FileHash -Algorithm SHA256 $path
  Write-Host "SHA256: $($hash.Hash)"
} catch { Write-Host "Get-FileHash error: $_" }
try {
  $bytes = [System.IO.File]::ReadAllBytes($path)
  $slice = $bytes[0..15]
  $hex = ($slice | ForEach-Object { '{0:X2}' -f $_ }) -join ' '
  Write-Host "First 16 bytes: $hex"
  if ($slice[0] -eq 0x4D -and $slice[1] -eq 0x5A) {
    Write-Host "Magic: MZ (likely PE file)"
  } else {
    Write-Host "Magic: not MZ — file may be corrupt or not a PE DLL"
  }
} catch { Write-Host "Read bytes error: $_" }
try {
  if (Get-Command dumpbin -ErrorAction SilentlyContinue) {
    dumpbin /headers $path
  } else {
    Write-Host "dumpbin not found in PATH — skip headers. Use Developer Command Prompt or VS tools."
  }
} catch { Write-Host "dumpbin error: $_" }
try {
  $sig = Get-AuthenticodeSignature $path
  Write-Host "Authenticode Signature Status: $($sig.Status)"
  $sig | Format-List
} catch { Write-Host "Get-AuthenticodeSignature error: $_" }
