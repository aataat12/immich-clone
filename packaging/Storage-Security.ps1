# Shared by the installer and disposable Windows security acceptance.
function Assert-TrustedCodeTree([string]$Path) {
    # Installer identity, administrators and SYSTEM are trusted code writers.
    # Unrelated local accounts and LocalService may read/execute, never modify code.
    # This is a stationary ACL/owner check; it does not close SEC-03 path races.
    Assert-NoReparsePath $Path
    $root = $null
    try { $root = Get-Item -LiteralPath ([IO.Path]::GetFullPath($Path)) -Force -ErrorAction Stop }
    catch [System.Management.Automation.ItemNotFoundException] { return }
    if (!$root.PSIsContainer) { throw 'Application code root must be a directory' }
    $trusted = @{'S-1-5-18'=$true; 'S-1-5-32-544'=$true}
    $trusted[[Security.Principal.WindowsIdentity]::GetCurrent().User.Value] = $true
    # Windows servicing owns/writes protected Program Files descendants.
    $trusted['S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464'] = $true
    $trustedOwners = @{} + $trusted
    $pending = New-Object 'System.Collections.Generic.Stack[System.IO.FileSystemInfo]'
    $pending.Push($root)
    while ($pending.Count -gt 0) {
        $item = $pending.Pop()
        if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'Application code tree contains a reparse point' }
        $acl = Get-Acl -LiteralPath $item.FullName -ErrorAction Stop
        if (!$trustedOwners.ContainsKey($acl.GetOwner([Security.Principal.SecurityIdentifier]).Value)) { throw 'Application code tree has an untrusted owner' }
        $raw = [Security.AccessControl.RawSecurityDescriptor]::new($acl.GetSecurityDescriptorBinaryForm(), 0)
        if (($raw.ControlFlags -band [Security.AccessControl.ControlFlags]::DiscretionaryAclPresent) -eq 0 -or $null -eq $raw.DiscretionaryAcl) { throw 'Application code tree has a null DACL' }
        foreach ($ace in $raw.DiscretionaryAcl) {
            if (!($ace -is [Security.AccessControl.CommonAce]) -or $ace.IsCallback -or $ace.AceQualifier -ne [Security.AccessControl.AceQualifier]::AccessAllowed) { throw 'Application code tree has an unsupported access rule' }
            # CREATOR OWNER inherit-only rules resolve to the trusted creating
            # installer identity; no untrusted principal can create code here.
            if ($ace.SecurityIdentifier.Value -eq 'S-1-3-0' -and $ace.AceFlags.HasFlag([Security.AccessControl.AceFlags]::InheritOnly)) { continue }
            # Include inherit-only grants: future installed children must be safe.
            # Write data/append/EA/attributes, delete/delete-child, DACL/owner and
            # generic write/all grant mutation, including on the code root itself.
            $mutation = ([int64]$ace.AccessMask -band 0xffffffffL) -band 0x500d0156L
            if ($mutation -ne 0 -and !$trusted.ContainsKey($ace.SecurityIdentifier.Value)) { throw 'Application code tree is writable by an untrusted identity' }
        }
        if ($item.PSIsContainer) {
            foreach ($child in [IO.Directory]::EnumerateFileSystemEntries($item.FullName)) {
                $pending.Push((Get-Item -LiteralPath $child -Force -ErrorAction Stop))
            }
        }
    }
}

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

function Get-PrivateDirectoryPolicy([string]$ReaderSid, [switch]$ServiceAccess) {
    $policy = @{
        'S-1-5-18' = [int64][int][Security.AccessControl.FileSystemRights]::FullControl
        'S-1-5-32-544' = [int64][int][Security.AccessControl.FileSystemRights]::FullControl
    }
    if ($ServiceAccess) { $policy['S-1-5-19'] = [int64][int][Security.AccessControl.FileSystemRights]::Modify }
    if ($ReaderSid -and !$policy.ContainsKey($ReaderSid)) { $policy[$ReaderSid] = [int64][int][Security.AccessControl.FileSystemRights]::ReadAndExecute }
    foreach ($sid in @($policy.Keys)) {
        # Windows adds Synchronize to normal Allow rules created from these rights.
        $policy[$sid] = $policy[$sid] -bor [int64][int][Security.AccessControl.FileSystemRights]::Synchronize
    }
    return $policy
}

function Assert-PrivateItemAcl([IO.FileSystemInfo]$Item, $Policy, $Owners, [switch]$VerifyInherited, [switch]$Root) {
    $acl = Get-Acl -LiteralPath $Item.FullName -ErrorAction Stop
    $owner = $acl.GetOwner([Security.Principal.SecurityIdentifier])
    if ($null -eq $owner -or !$Owners.ContainsKey($owner.Value)) { throw 'Application-owned tree has an unexpected owner' }
    $raw = [Security.AccessControl.RawSecurityDescriptor]::new($acl.GetSecurityDescriptorBinaryForm(), 0)
    if (($raw.ControlFlags -band [Security.AccessControl.ControlFlags]::DiscretionaryAclPresent) -eq 0 -or $null -eq $raw.DiscretionaryAcl) { throw 'Application-owned tree has a null DACL' }
    # Preflight admits root grants that replacement can fix. Verification never
    # assumes inheritance succeeded: root and descendants use the full policy.
    if ($Root -and !$VerifyInherited) { return }
    if ($Root -and !$acl.AreAccessRulesProtected) { throw 'Application-owned root is not protected against parent inheritance' }
    foreach ($ace in $raw.DiscretionaryAcl) {
        if ($ace.AceFlags.HasFlag([Security.AccessControl.AceFlags]::Inherited) -and !$acl.AreAccessRulesProtected -and !$VerifyInherited) { continue }
        # Higher-level GetAccessRules may omit callback/object ACE forms.
        if (!($ace -is [Security.AccessControl.CommonAce]) -or $ace.IsCallback -or $ace.AceQualifier -ne [Security.AccessControl.AceQualifier]::AccessAllowed) { throw 'Application-owned tree has an unsupported descendant access rule' }
    }
    $selfRights = @{}
    $childRights = @{}
    foreach ($rule in $acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier])) {
        if ($rule.IsInherited -and !$acl.AreAccessRulesProtected -and !$VerifyInherited) { continue }
        $sid = $rule.IdentityReference.Value
        $rights = [int64][int]$rule.FileSystemRights
        if ($rule.AccessControlType -ne [Security.AccessControl.AccessControlType]::Allow -or !$Policy.ContainsKey($sid) -or $rights -lt 0 -or ($rights -band (-bnot $Policy[$sid])) -ne 0) { throw 'Application-owned tree has an unsafe descendant access rule' }
        if (($rule.PropagationFlags -band [Security.AccessControl.PropagationFlags]::InheritOnly) -eq 0) { $selfRights[$sid] = [int64]$selfRights[$sid] -bor $rights }
        if ($rule.PropagationFlags -eq [Security.AccessControl.PropagationFlags]::None -and ($rule.InheritanceFlags -band [Security.AccessControl.InheritanceFlags]'ContainerInherit,ObjectInherit') -eq [Security.AccessControl.InheritanceFlags]'ContainerInherit,ObjectInherit') { $childRights[$sid] = [int64]$childRights[$sid] -bor $rights }
    }
    if ($acl.AreAccessRulesProtected -or $VerifyInherited) {
        foreach ($sid in $Policy.Keys) {
            if (([int64]$selfRights[$sid] -band $Policy[$sid]) -ne $Policy[$sid]) { throw 'Application-owned descendant lacks required access' }
            if ($Item -is [IO.DirectoryInfo] -and ([int64]$childRights[$sid] -band $Policy[$sid]) -ne $Policy[$sid]) { throw 'Application-owned directory lacks required inheritance' }
        }
    }
}

function Assert-PrivateDirectoryTree([string]$Path, [string]$ReaderSid, [switch]$ServiceAccess, [string]$OwnerSid, [switch]$VerifyInherited) {
    # Read-only stationary-tree audit. Enumeration never recurses through a link;
    # every child is inspected before it may be pushed onto the directory stack.
    # Concurrent path substitution is a distinct, unresolved handle/identity gate.
    Assert-NoReparsePath $Path
    $root = $null
    try { $root = Get-Item -LiteralPath ([IO.Path]::GetFullPath($Path)) -Force -ErrorAction Stop }
    catch [System.Management.Automation.ItemNotFoundException] { return }
    if (!$root.PSIsContainer) { throw 'Application-owned storage root must be a directory' }
    $policy = Get-PrivateDirectoryPolicy -ReaderSid $ReaderSid -ServiceAccess:$ServiceAccess
    $owners = @{}
    foreach ($sid in $policy.Keys) { $owners[$sid] = $true }
    if (!$OwnerSid) { $OwnerSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value }
    $owners[$OwnerSid] = $true
    Assert-PrivateItemAcl -Item $root -Policy $policy -Owners $owners -VerifyInherited:$VerifyInherited -Root
    $pending = New-Object 'Collections.Generic.Stack[string]'
    $pending.Push($root.FullName)
    while ($pending.Count -gt 0) {
        $directory = $pending.Pop()
        foreach ($path in [IO.Directory]::EnumerateFileSystemEntries($directory)) {
            $item = Get-Item -LiteralPath $path -Force -ErrorAction Stop
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'Application-owned tree contains a reparse point' }
            Assert-PrivateItemAcl -Item $item -Policy $policy -Owners $owners -VerifyInherited:$VerifyInherited
            if ($item.PSIsContainer) { $pending.Push($item.FullName) }
        }
    }
}
