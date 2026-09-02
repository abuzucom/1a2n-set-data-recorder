# Template Drift

## Source

This repository adopts `abuzucom/agents` commit
`64bc55e17905720e9c6ba505326c49f802371cfb`.

## Expected Differences

- `AGENTS.md` replaces the upstream source-repository orientation with verified
  facts for this planned Go and Gin server.
- `.claudeignore` excludes only verified secret and repository paths. The
  upstream Node and Next.js exclusions do not match this repository.
- `Makefile` and `sync-check.yml` scan this repository's local drift record.
- `DRIFT.md` links to this record instead of listing upstream adopters.
- `CHANGELOG.md` records this repository's changes instead of upstream history.
- `AGENTS.md` removes Markdown hard-break whitespace from examples so staged
  validation with `git diff --check` passes. Synchronized copies carry the
  same formatting.

## Optional Templates

- `CONTRIBUTING.md.example`, `SECURITY.md.example`, and
  `plan/HANDOFF.md.example` remain templates. No live copies exist.
- GitHub issue and pull request templates remain available under `.github`.

## Upstream Record

The upstream adopter record and drift issue remain pending hosted GitHub
authorization.
