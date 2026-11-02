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

function Assert-PrivateDirectoryTree([string]$Path, [string]$ReaderSid, [switch]$ServiceAccess, [string]$OwnerSid) {
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
    $rootAcl = Get-Acl -LiteralPath $root.FullName -ErrorAction Stop
    $rootRaw = [Security.AccessControl.RawSecurityDescriptor]::new($rootAcl.GetSecurityDescriptorBinaryForm(), 0)
    if (($rootRaw.ControlFlags -band [Security.AccessControl.ControlFlags]::DiscretionaryAclPresent) -eq 0 -or $null -eq $rootRaw.DiscretionaryAcl) { throw 'Application-owned tree has a null DACL' }
    $rootOwner = $rootAcl.GetOwner([Security.Principal.SecurityIdentifier])
    if ($null -eq $rootOwner -or !$owners.ContainsKey($rootOwner.Value)) { throw 'Application-owned tree has an unexpected owner' }
    $pending = New-Object 'Collections.Generic.Stack[string]'
    $pending.Push($root.FullName)
    while ($pending.Count -gt 0) {
        $directory = $pending.Pop()
        foreach ($path in [IO.Directory]::EnumerateFileSystemEntries($directory)) {
            $item = Get-Item -LiteralPath $path -Force -ErrorAction Stop
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'Application-owned tree contains a reparse point' }
            $acl = Get-Acl -LiteralPath $path -ErrorAction Stop
            $owner = $acl.GetOwner([Security.Principal.SecurityIdentifier])
            if ($null -eq $owner -or !$owners.ContainsKey($owner.Value)) { throw 'Application-owned tree has an unexpected owner' }
            $raw = [Security.AccessControl.RawSecurityDescriptor]::new($acl.GetSecurityDescriptorBinaryForm(), 0)
            if (($raw.ControlFlags -band [Security.AccessControl.ControlFlags]::DiscretionaryAclPresent) -eq 0 -or $null -eq $raw.DiscretionaryAcl) { throw 'Application-owned tree has a null DACL' }
            # GetAccessRules can omit callback/object ACE forms. Fail closed on
            # such noncanonical explicit/protected DACL entries rather than
            # accidentally treating an unrepresented allow as safe.
            foreach ($ace in $raw.DiscretionaryAcl) {
                if ($ace.AceFlags.HasFlag([Security.AccessControl.AceFlags]::Inherited) -and !$acl.AreAccessRulesProtected) { continue }
                if (!($ace -is [Security.AccessControl.CommonAce]) -or $ace.IsCallback -or $ace.AceQualifier -ne [Security.AccessControl.AceQualifier]::AccessAllowed) { throw 'Application-owned tree has an unsupported descendant access rule' }
            }
            $selfRights = @{}
            $childRights = @{}
            foreach ($rule in $acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier])) {
                # For inheriting descendants, old inherited rights disappear when
                # the root is replaced. Explicit rules on intermediate ancestors
                # are audited separately. Protected DACLs receive no root repair.
                if ($rule.IsInherited -and !$acl.AreAccessRulesProtected) { continue }
                $sid = $rule.IdentityReference.Value
                $rights = [int64][int]$rule.FileSystemRights
                if ($rule.AccessControlType -ne [Security.AccessControl.AccessControlType]::Allow -or !$policy.ContainsKey($sid) -or $rights -lt 0 -or ($rights -band (-bnot $policy[$sid])) -ne 0) { throw 'Application-owned tree has an unsafe descendant access rule' }
                if (($rule.PropagationFlags -band [Security.AccessControl.PropagationFlags]::InheritOnly) -eq 0) { $selfRights[$sid] = [int64]$selfRights[$sid] -bor $rights }
                if ($rule.PropagationFlags -eq [Security.AccessControl.PropagationFlags]::None -and ($rule.InheritanceFlags -band [Security.AccessControl.InheritanceFlags]'ContainerInherit,ObjectInherit') -eq [Security.AccessControl.InheritanceFlags]'ContainerInherit,ObjectInherit') { $childRights[$sid] = [int64]$childRights[$sid] -bor $rights }
            }
            if ($acl.AreAccessRulesProtected) {
                foreach ($sid in $policy.Keys) {
                    if (([int64]$selfRights[$sid] -band $policy[$sid]) -ne $policy[$sid]) { throw 'Application-owned protected descendant lacks required access' }
                    if ($item.PSIsContainer -and ([int64]$childRights[$sid] -band $policy[$sid]) -ne $policy[$sid]) { throw 'Application-owned protected directory lacks required inheritance' }
                }
            }
            if ($item.PSIsContainer) { $pending.Push($item.FullName) }
        }
    }
}
