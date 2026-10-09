# SDE converter

Vendored from EVEShipFit/sde-patched at v12.3579973.0,
commit `2ee8ea5334ed5f90da800fddc90a6fab0d6c2178`.
https://github.com/EVEShipFit/sde-patched

The sde, patch and fbs implementations and regression tests are retained.
Go import paths are rewritten under Shrike's internal package, and three generated
Go files are normalized with gofmt. The schema, patch rules and conversion logic
are unchanged. The CLI/web editor is omitted.
The license, schemas and patches are retained. Root dogmadata wraps the existing
cached CCP archive loader and stages a complete replacement bundle before rename.

Shrike's lint configuration excludes upstream style/unused findings and reviewed
public-data permission/file-path warnings in this vendored directory. Other
security checks, compiler checks and upstream regression tests remain enabled;
the application wrapper follows the normal repository lint rules.
