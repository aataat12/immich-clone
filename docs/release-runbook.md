# Immich Clone release runbook

This is the authoritative procedure for producing a release. Run it from the
persistent `Immich Clone — Release Production` discussion.

## 1. Initialize the release

1. Read `AGENTS.md`, this file, `release-state.json`, the latest release notes,
   and the applicable workflows under `.github/workflows/`.
2. Confirm the intended version, predecessor tag, canonical source checkout,
   release branch, and acceptance workflow.
3. Run `scripts/Release-Preflight.ps1` and record only its sanitized result.
4. Create or adapt the release notes, acceptance manifest, workflow, and
   publishing script. Keep version, branch, tag, run, job, artifact, and asset
   values explicit and reviewable.

## 2. Build and validate in the cloud

1. Push the intended source and workflow to the release branch. A push trigger
   starts GitHub Actions; no local Go, Inno Setup, browser, or GitHub CLI is
   required.
2. Monitor the workflow through GitHub REST or the connected GitHub tools.
3. Require the acceptance run to prove that its checkout SHA equals the intended
   source commit and that all required jobs succeed.
4. Download the exact accepted artifact and verify the ZIP digest, installer
   digest, file sizes, manifest, and evidence files. Do not rebuild after this.

## 3. Publish the exact tested bytes

1. Review the artifact and acceptance manifest for version, source SHA, run ID,
   job IDs, artifact ID, and SHA-256 values.
2. Run the parameterized publishing script in verification mode first.
3. Publish only after verification succeeds. The tag must point to the tested
   source commit, and the release upload must use the exact tested installer
   bytes.
4. Verify the remote release, tag target, asset names, sizes, and digests.
5. Copy the exact published installer to the required Dropbox delivery folder
   and hash-check the destination.

## 4. Record and hand off

Update `docs/release-state.json` with sanitized evidence and known limits. State
which gates were run on GitHub-hosted Windows and which remain unverified, such
as Windows 10, clean-machine, reboot, endurance, or code-signing coverage.

## Access rules

GitHub CLI installation is not part of this process. If GitHub connector writes
are unavailable, use GitHub REST and the bundled Git executable with the user's
existing Windows Credential Manager sign-in. A missing `gh` command or a
restricted-context `SEC_E_NO_CREDENTIALS` error is diagnostic information, not a
release conclusion. Retry the appropriate user-context route without printing
the credential or token.

Never place tokens, private machine reports, personal filesystem details, or raw
credential output in the repository, Actions artifacts, screenshots, or release
notes.
