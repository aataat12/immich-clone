# Shared by the installer and disposable Windows security acceptance.
function Assert-NoReparsePath([string]$Path) {
    $current = [IO.Path]::GetFullPath($Path)
    while ($current) {
        $item = $null
        try { $item = Get-Item -LiteralPath $current -Force -ErrorAction Stop }
        catch [System.Management.Automation.ItemNotFoundException] { }
        if ($null -ne $item) {
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
                throw 'Installation paths must not traverse reparse points'
            }
        }
        $parent = [IO.Directory]::GetParent($current)
        if ($null -eq $parent) { break }
        $current = $parent.FullName
    }
}

function Set-PrivateDirectoryAcl([string]$Path, [string]$ReaderSid, [switch]$ServiceAccess) {
    Assert-NoReparsePath $Path
    [IO.Directory]::CreateDirectory($Path) | Out-Null
    Assert-NoReparsePath $Path
    # Build a fresh DACL: grant:r only replaces named principals and can retain
    # unrelated explicit allow/deny entries on a preexisting directory.
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetAccessRuleProtection($true, $false)
    $grants = @{'S-1-5-18' = 'FullControl'; 'S-1-5-32-544' = 'FullControl'}
    if ($ServiceAccess) { $grants['S-1-5-19'] = 'Modify' }
    if ($ReaderSid -and !$grants.ContainsKey($ReaderSid)) { $grants[$ReaderSid] = 'ReadAndExecute' }
    foreach ($sid in $grants.Keys) {
        $rule = New-Object Security.AccessControl.FileSystemAccessRule(
            (New-Object Security.Principal.SecurityIdentifier($sid)),
            [Security.AccessControl.FileSystemRights]$grants[$sid],
            [Security.AccessControl.InheritanceFlags]'ContainerInherit,ObjectInherit',
            [Security.AccessControl.PropagationFlags]::None,
            [Security.AccessControl.AccessControlType]::Allow)
        $acl.AddAccessRule($rule)
    }
    Set-Acl -LiteralPath $Path -AclObject $acl
}
