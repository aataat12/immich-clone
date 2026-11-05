# SEC-03 separate-account native metadata checkpoint

This advances the original installer/metadata concurrency row; it does not close
that row or SEC-03. Accepted interrupted-upgrade behavior remains the baseline.

## Executed evidence

Windows Server2022 [run41953634876](https://github.com/aataat12/immich-clone/actions/runs/41953634876),
source `f4b8098b81879a07c453e57cc51b0a5ab89e7083`, job127003195293 completed success.
The native installerfiles suite, helper/test-executable build, module verification,
vet, script parser and corrected separate-account harness passed. Full job logs
and terminal run conclusion were inspected. The runner canonicalized the two Go
sources before validation; the saved checkpoint retains that exact formatting.

A real separately logged-on standard account first demonstrated successful file
write, rename, outgoing hardlink creation and an actual new explicit ACL grant in
a permissive control. Against selected private transaction metadata it attempted
those operations continuously during native begin/audit, actual complete payload
publication and commit. Attempt counters advanced while native processes were
active; twelve audits completed under the actor. The sampled audit phase recorded
zero successful mutations,210 refusals,94 unavailable/missing-name results and no
unexpected exceptions. Subsequent publication/retirement phases also required zero
successful mutations and no unclassified failures.

The preauthorized-writer fixture deliberately grants access before a standard
account opens a writable handle, then establishes/verifies the exact private
BA/SYSTEM policy and unchanged owner/bytes. Native audit refuses while that old
capability remains open; release permits retry, preserving the hardened descriptor
and bytes. A preauthorized controller hash observer uses compatible sharing only
in the fixture; product sharing remains strict. This avoids confusing Windows
ACL normalization during intentional hardening with product-side mutation.

A compiled package test supplies a deterministic barrier after every selected
retirement leaf has been held/audited, before deletion. The controller observes
separate-account terminal replacement attempts at that barrier, then releases it.
The test interrupts after the first successful deletion, verifies remaining files
and valid committed terminal metadata, and retries through RecoverUpgrade. Cleanup
finishes without rolling back candidate bytes; terminal is deleted last, and the
accepted empty root/payload directories remain. Synthetic outside/catalog/access-
code bytes and descriptors remain unchanged; post-commit recovery is idempotent.

## Handoff corrections and traceability

The delivered worker script had no permissive ACL setup, unquoted spaced paths,
a stale reused marker, actor completion before native audit, an assertion allowing
missing pending metadata, a fabricated completion record and a root-removal
expectation contrary to the accepted lifecycle. The corrected harness uses actual
Windows impersonation and real native payload publication, bounded/reaped test
processes and a test-only held deletion barrier. There is no production flag,
environment bypass, listener, account, provider or dependency addition.

Intermediate failed gates remain visible:41951890781/41952202112 rejected exact
Framework NativeObjectSecurity numeric status wrappers that initially were not
classified; all protected mutation success counters were zero. Only the precise
numeric wrapper is decoded now; arbitrary exceptions still fail.41952635174 and
41952927940 exposed unreliable fixture ACL-restoration assumptions.41953277479
exposed the default hash reader's sharing conflict with the intentional writer.
The final fixture verifies private hardening before measuring preservation, with
an explicit controller observer and unchanged strict native refusal assertions.
All final assertions passed in41953634876; no failed installer was published.

## Remaining original scope

This is native action/retirement execution, not actual Inno Setup/rollback/uninstall
metadata replacement acceptance. The fixture uses synthetic text payload/catalog
sentinels and creates no application service; it does not establish real library,
SCM, genuine prior-release, full-provider, arbitrary ancestor-replacement or every
action-to-action boundary result. Existing held-handle protections are not claimed
to span the whole Inno/PowerShell transaction. The active Inno journal/image audit
exception remains narrow and needs its own integrated concurrent proof.

Next is actual Setup metadata/action-boundary protection and separate-account
interleaving evidence, followed by the existing genuine-release/full-provider/
recovery combined gate. Those are the same two outstanding rows; no forecast
reduction or closure is claimed from this checkpoint. Published0.1.39, security
4 closed/4 open and original Group3 deferral remain unchanged. No PC action, new
product dependency/provider/schema/version/release/cost or Windows10 certification.
