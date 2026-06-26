#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Esegue la diagnostica CP/M 8080EXM (o un'altra ROM) sull'emulatore retronet-8080.

.DESCRIPTION
  Imposta le variabili d'ambiente attese dal test TestCPMDiagnosticROM e lo lancia.
  Per default usa il backend ALU "native" (operatori Go): 8080EXM gira in pochi
  secondi. Con -Alu gate gira sull'ALU a porte logiche (alcuni minuti). I due
  backend sono garantiti identici dal test differenziale, quindi l'esito coincide.

  Le variabili d'ambiente preesistenti vengono salvate e ripristinate a fine corsa.

.PARAMETER Rom
  Percorso del .COM/.bin diagnostico. Se omesso, usa $env:RETRONET_8080_DIAG_ROM
  oppure cerca un file *EXM*.COM in conformance\testdata\diag\.

.PARAMETER Alu
  Backend ALU: 'native' (default, veloce) o 'gate' (porte logiche).

.PARAMETER MaxSteps
  Limite di step di esecuzione; 0 = nessun limite (default).

.EXAMPLE
  .\scripts\run-exm.ps1 -Rom C:\rom\8080EXM.COM

.EXAMPLE
  .\scripts\run-exm.ps1 -Alu gate          # stessa ROM, ALU a porte (confronto)
#>
[CmdletBinding()]
param(
    [string]$Rom = $env:RETRONET_8080_DIAG_ROM,
    [ValidateSet('native', 'gate')][string]$Alu = 'native',
    [int]$MaxSteps = 0
)

$ErrorActionPreference = 'Stop'

# Radice del repo = cartella genitrice di questo script (scripts\).
$repoRoot = Split-Path -Parent $PSScriptRoot

# Trova la ROM se non indicata: cerca un *EXM*.COM nella cartella diagnostiche.
if ([string]::IsNullOrWhiteSpace($Rom)) {
    $diagDir = Join-Path $repoRoot 'conformance\testdata\diag'
    $candidate = Get-ChildItem -Path $diagDir -Filter '*EXM*.COM' -File -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if ($candidate) { $Rom = $candidate.FullName }
}

if ([string]::IsNullOrWhiteSpace($Rom) -or -not (Test-Path -LiteralPath $Rom)) {
    Write-Error ("ROM non trovata. Passa -Rom <percorso\8080EXM.COM> " +
        "oppure mettila in conformance\testdata\diag\ (gitignored).")
    exit 1
}
$Rom = (Resolve-Path -LiteralPath $Rom).Path

Write-Host "ROM      : $Rom"
Write-Host "Backend  : $Alu"
Write-Host "MaxSteps : $MaxSteps"
Write-Host ''

# Salva l'ambiente precedente e imposta quello atteso dal test.
$saved = @{
    RETRONET_8080_ALU           = $env:RETRONET_8080_ALU
    RETRONET_8080_DIAG_ROM      = $env:RETRONET_8080_DIAG_ROM
    RETRONET_8080_DIAG_MAXSTEPS = $env:RETRONET_8080_DIAG_MAXSTEPS
}
try {
    $env:RETRONET_8080_ALU           = $Alu
    $env:RETRONET_8080_DIAG_ROM      = $Rom
    $env:RETRONET_8080_DIAG_MAXSTEPS = "$MaxSteps"

    Push-Location $repoRoot
    try {
        go test ./conformance/ -run TestCPMDiagnosticROM -v -timeout 0
        exit $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
}
finally {
    # Ripristina le variabili d'ambiente preesistenti.
    foreach ($k in $saved.Keys) {
        if ($null -eq $saved[$k]) {
            Remove-Item "Env:$k" -ErrorAction SilentlyContinue
        }
        else {
            Set-Item "Env:$k" $saved[$k]
        }
    }
}
