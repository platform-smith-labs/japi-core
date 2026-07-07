---
type: gotcha
title: "DB credentials come from the repo's .env via the build script — never -e flags"
tags: [build, docker, credentials, configuration]
timestamp: 2026-07-07T01:02:42Z
description: "The build script injects database credentials from .env at container start; the house rule is to change credentials by editing .env and re-running the script, not by passing -e DB_* at docker run"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - build.sh
  - Dockerfile
  - CLAUDE.md
---
The migrator image itself contains **no credentials** — only the binary and the migration
files. When run through the repo's build script, database credentials are injected at
**container start** from the repo's `.env` file (supplied as the container's env file). The
image is always built and run through that script — never with a hand-rolled `docker build`
/ `docker run`.

The trap: pointing an already-built image at a different database by passing ad-hoc
`-e DB_*` flags to `docker run`. Mechanically this would work, but the house rule forbids
it — credential or connection changes are made by **editing `.env` and re-running the build
script** (which also runs the migrations when asked), so the env file stays the single
source of truth.

Note this applies to the standalone/local image workflow; orchestrated environments
(compose, Kubernetes) inject their own DB environment through their manifests instead.
Beware that some in-repo prose describes the credentials as "baked in at build time" —
the run-time env-file injection above is the code-true mechanism.
