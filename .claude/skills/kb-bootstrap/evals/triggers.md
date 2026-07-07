# Description trigger evals — /kb-bootstrap

The skill is `disable-model-invocation` (human/playbook-invoked), so triggering is by explicit intent.
These check the description reads correctly for the intended invocations and excludes the wrong ones.

## SHOULD invoke (a human/playbook asking for KB generation)
1. "Bootstrap the knowledge base for this repo."
2. "Generate docs/kb for orchestrator."
3. "Refresh this repo's KB / project brief."
4. "Regenerate the capability summaries for peers."
5. "Create a cross-repo knowledge base for this service."
6. "Update the KB after these changes."

## SHOULD NOT invoke
1. "Explain how this function works." (code Q&A, not KB generation)
2. "Write exhaustive API docs for every endpoint." (exhaustive per-symbol docs — out of scope §0)
3. "Document this one function's parameters." (per-function docs — out of scope)
4. "Summarize this file." (not a repo KB)
