cls
# Read-only; Windows PowerShell 5.1; no elevation or dependencies required.
$ErrorActionPreference = 'Stop'
function Read-Optional([scriptblock]$Query) {
    try { & $Query } catch { 'Unavailable' }
}
$os = Get-CimInstance Win32_OperatingSystem
$cv = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion'
$report = [ordered]@{
    Windows = [ordered]@{
        Edition = $os.Caption
        EditionId = $cv.EditionID
        DisplayVersion = $cv.DisplayVersion
        Version = $os.Version
        Build = ('{0}.{1}' -f $cv.CurrentBuildNumber, $cv.UBR)
        OSArchitecture = $os.OSArchitecture
        ProcessArchitecture = $env:PROCESSOR_ARCHITECTURE
        PowerShellVersion = $PSVersionTable.PSVersion.ToString()
        ESUEnrollment = 'Please confirm manually; not inferred from installed updates'
    }
    CPU = @(Get-CimInstance Win32_Processor | Select-Object Name, Architecture, AddressWidth, NumberOfCores, NumberOfLogicalProcessors)
    RAMGiB = [math]::Round((Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory / 1GB, 1)
    GPU = @(Read-Optional { Get-CimInstance Win32_VideoController | Select-Object Name, DriverVersion })
    FixedVolumes = @(Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | ForEach-Object {
        [ordered]@{ Drive = $_.DeviceID; FileSystem = $_.FileSystem
            SizeGiB = [math]::Round($_.Size / 1GB, 1)
            FreeGiB = [math]::Round($_.FreeSpace / 1GB, 1) }
    })
    PhysicalStorage = @(Read-Optional {
        Get-PhysicalDisk | Select-Object MediaType, BusType, @{Name='SizeGiB';Expression={[math]::Round($_.Size / 1GB, 1)}}
    })
}
$report | ConvertTo-Json -Depth 6
Write-Host "`nAlso tell me: intended media drive; ESU status if known; existing eligible Visual Studio/Azure dev-test or licensed cloud Windows 10 access."
Write-Host 'Review the output, then paste it into this conversation. No file was saved or transmitted.'
