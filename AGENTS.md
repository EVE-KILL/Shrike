# Agent instructions

## Releases

Read `docs/deployment/releases.md` before you create a release.

After you commit and push a release change to `main`, do not wait locally for CI
and do not create the tag locally. Start the promotion workflow immediately:

```bash
gh workflow run release.yml --ref main -f version=vMAJOR.MINOR.PATCH
```

The workflow pins the current `main` commit, waits for that commit's push CI,
creates the annotated tag only after CI passes, and publishes the multi-platform
image. Watch that single workflow through image publication. A release is not
published merely because the tag exists.

Manual tag pushes are a recovery path only. Never move or reuse a published tag.
