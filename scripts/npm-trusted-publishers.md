# npm trusted-publisher setup (release prerequisite)

The `npm-publish` job in [`.github/workflows/release.yml`](../.github/workflows/release.yml)
publishes the npm acquisition channel (spec 037) with **GitHub OIDC trusted
publishing** — no long-lived `NPM_TOKEN` is stored anywhere. For that to work,
each of the five published package names must be registered **once** on
npmjs.com as trusting this repository's release workflow. This is a one-time
maintainer ops step, in the spirit of
[`setup-branch-protection.sh`](./setup-branch-protection.sh) — until it is done,
the first `npm publish` for a name fails with an authentication error.

There is no npm CLI for configuring trusted publishers; it is done through the
npmjs.com web UI, so this is documentation rather than a script.

## Packages

All five live under the `@luscii-healthtech` org scope and publish at the release
version (the tag without its leading `v`):

- `@luscii-healthtech/glassfrog` (the umbrella — exposes the `glassfrog` command)
- `@luscii-healthtech/glassfrog-darwin-arm64`
- `@luscii-healthtech/glassfrog-darwin-x64`
- `@luscii-healthtech/glassfrog-linux-arm64`
- `@luscii-healthtech/glassfrog-linux-x64`

## One-time setup (per package name)

Prerequisite: you are a member of the `@luscii-healthtech` npm org with publish
rights, and the org allows public scoped packages.

For **each** of the five package names above:

1. Sign in to <https://www.npmjs.com> as an org member.
2. Open the package's **Settings → Publishing access** (for a name that does not
   exist yet, configure the trusted publisher first — npm creates the package on
   the first trusted-publishing publish).
3. Add a **GitHub Actions** trusted publisher with:
   - **Organization / repository**: `Luscii/cli-glassfrog`
   - **Workflow filename**: `release.yml`
   - **Environment**: _(leave empty — the job uses no GitHub Environment)_
4. Save.

Once all five are registered, publishing a GitHub Release runs `release.yml`,
which builds + verifies the binaries and then publishes the platform packages
followed by the umbrella over OIDC, attaching provenance automatically.

## Verifying

After the first successful release, each package page on npmjs.com shows a
**Provenance** badge linking back to the `release.yml` run that published it. The
umbrella's `optionalDependencies` pin each platform package to the exact release
version (`=X.Y.Z`).
