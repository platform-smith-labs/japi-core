# Generation evals — assertions on a produced docs/kb bundle

Run `/kb-bootstrap` on a target repo, then assert:

1. **Lint-clean**: `kb-lint.sh <repo>` exits 0 (no FAIL).
2. **No source pointers**: no concept body contains a `file:line` or source-file link.
3. **Capability coverage**: `self/capabilities/` covers the repo's material peer-facing functionalities
   (spot-checked by a reviewer against the real surface).
4. **Brevity**: every capability ≤ ~120 lines; whole self/ narrative ≤ ~15k tokens.
5. **Honesty**: ungroundable facts appear as `UNKNOWN`, not invented (seed one hallucination and
   confirm VERIFY/lint surface it).
6. **Idempotence**: a second immediate run yields no semantic diff; `notes/`,`peers/`,`kb-config.yaml`
   survive.
