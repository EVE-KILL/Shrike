# Release pipeline

The release pipeline verifies Shrike and publishes one unified container.
It does not change Helm or deploy production.

## Continuous integration

Pull requests and pushes to `main` run the `CI` workflow.

The workflow runs three lanes in parallel.

The Go lane checks formatting and the generated OpenAPI contract. It runs
`go vet`, creates fresh Goose and River schemas, and runs the Go tests.

The frontend lane runs the Nuxt and dogma tests and type-checks the Nuxt
application.

The image lane builds the unified container and runs its version command. The
container build includes the Nuxt production build.

The workflow does not publish an image.

Configure branch protection to require the `CI / Verify source and unified
image` check before merging.

## Create a release

Release tags use Semantic Versioning with a `v` prefix.

1. Update local `main`.
2. Confirm that CI passed for the release commit.
3. Create an annotated tag.
4. Push the tag.

```bash
git switch main
git pull --ff-only
git tag -a v1.0.0 -m "Shrike v1.0.0"
git push origin v1.0.0
```

The `Release` workflow accepts stable tags and prerelease tags.

Examples:

```text
v1.0.0
v1.1.0-rc.1
```

The release commit must exist on `main`.

The workflow confirms that the tagged commit passed the `main` CI workflow. It
does not rerun CI.

Stable releases update `latest`. Prereleases do not update `latest`.

It publishes these GitHub Container Registry tags:

```text
ghcr.io/eve-kill/shrike:v1.0.0
ghcr.io/eve-kill/shrike:1.0.0
ghcr.io/eve-kill/shrike:1.0
ghcr.io/eve-kill/shrike:1
ghcr.io/eve-kill/shrike:sha-COMMIT
ghcr.io/eve-kill/shrike:latest
```

Each image supports `linux/amd64` and `linux/arm64`.

The workflow also publishes a software bill of materials and provenance.

## Deploy a release

Set the Helm image tag or digest in a separate deployment change.

Review that change before deployment.

Do not make the release workflow edit Helm values.

Do not make the release workflow contact the production cluster.

## Verification

1. Open the tag run in GitHub Actions.
2. Confirm that the `Publish unified image` job passed.
3. Read the published digest from the workflow summary.
4. Inspect both image platforms.

```bash
docker buildx imagetools inspect ghcr.io/eve-kill/shrike:v1.0.0
```

Run the released image before you update Helm:

```bash
docker run --rm ghcr.io/eve-kill/shrike:v1.0.0 version
```

## Recovery

Do not move or reuse a published release tag.

Create a new patch release when a released image is defective.

Keep production on the previous image until the replacement passes
verification.

Restore the previous Helm image tag or digest when production already uses the
defective image.
