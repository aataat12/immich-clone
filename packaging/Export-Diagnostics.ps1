param([string]$OutputPath)
$ErrorActionPreference = 'Stop'
# No machine inventory, transcript, registry dump or raw setup logs are collected.
$root = Join-Path $env:ProgramData 'ImmichCloneDiagnostics'
if (!$OutputPath) {
    Add-Type -AssemblyName System.Windows.Forms
    $dialog = New-Object System.Windows.Forms.SaveFileDialog
    $dialog.FileName = 'ImmichClone-diagnostics.log'
    $dialog.Filter = 'Diagnostic log (*.log)|*.log'
    $dialog.InitialDirectory = [Environment]::GetFolderPath('Desktop')
    if ($dialog.ShowDialog() -ne [System.Windows.Forms.DialogResult]::OK) { exit 0 }
    $OutputPath = $dialog.FileName
    $dialog.Dispose()
}
$builder = New-Object Text.StringBuilder
$null = $builder.AppendLine('Immich Clone diagnostic log. No hardware inventory, paths, media names, hashes, credentials or request contents are collected.')
foreach ($name in @('installer.jsonl.previous','installer.jsonl','application.jsonl.previous','application.jsonl','commands.jsonl.previous','commands.jsonl')) {
    $file = Join-Path $root $name
    if (Test-Path -LiteralPath $file) {
        $null = $builder.AppendLine("`n--- $name ---")
        # Read with sharing so export also works while the service is running.
        $stream = [IO.File]::Open($file,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::ReadWrite -bor [IO.FileShare]::Delete)
        try {
            if ($stream.Length -gt 4MB) { throw 'Diagnostic file exceeds its expected size limit' }
            $reader = New-Object IO.StreamReader($stream)
            try { $null = $builder.AppendLine($reader.ReadToEnd()) } finally { $reader.Dispose() }
        } finally { $stream.Dispose() }
    }
}
$fallback = Join-Path $PSScriptRoot 'install-diagnostic.txt'
if (Test-Path -LiteralPath $fallback) {
    $null = $builder.AppendLine("`n--- installer fallback (stage/type/code only) ---")
    $null = $builder.AppendLine((Get-Content -LiteralPath $fallback -Raw))
}
[IO.File]::WriteAllText($OutputPath,$builder.ToString(),(New-Object Text.UTF8Encoding($false)))
Write-Host 'Diagnostic log saved. Review it and attach it privately in this conversation.'
