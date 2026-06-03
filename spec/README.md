# GlassFrog API v5 specification

This directory holds a vendored copy of the GlassFrog API v5 OpenAPI specification. It is the authoritative contract this CLI implements — see [CONSTITUTION.md](../CONSTITUTION.md), Principle I (Spec Fidelity).

|  |  |
|---|---|
| **Source** | https://app.glassfrog.com/api/v5/docs/spec.yaml |
| **File** | `glassfrog-api-v5.yaml` |
| **Title** | GlassFrog API v5 (Beta) |
| **Spec version** | `5.0.0` (`info.version`) |
| **OpenAPI** | 3.0.3 |
| **First vendored** | 2026-06-03 |

## Why it's here

The spec is a fundamental document for this project: every command must map to an operation it defines. Vendoring it into the repository means the contract travels with the code, and its changes are tracked in git history.

## Beta — expect changes

The v5 API is in Beta and the spec can change. To track a change:

1. Run `./scripts/refresh-spec.sh` to re-fetch the spec into this directory.
2. Review the diff: `git diff -- spec/glassfrog-api-v5.yaml`.
3. Commit it with a message describing what moved in the API. Each refresh that differs becomes a tracked commit, so **git history is the spec changelog**.

Update the **Spec version** / **First vendored** fields above if `info.version` changes.
