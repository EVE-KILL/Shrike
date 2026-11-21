# SDE converter

Vendored from EVEShipFit/sde-patched at v14.3720952.0,
commit `d6b270f9b7121b25e0820ec9d4037f593beb7952`.
https://github.com/EVEShipFit/sde-patched

The sde, patch and fbs implementations and regression tests are retained.
Go import paths use Shrike's internal package. The generated Go files are
formatted with gofmt. The vendor includes the schema, patch rules and conversion
logic.
The license, schemas and patches are retained. Root dogmadata wraps the existing
cached CCP archive loader and stages a complete replacement bundle before rename.

Shrike's lint configuration excludes upstream style/unused findings and reviewed
public-data permission/file-path warnings in this vendored directory. Other
security checks, compiler checks and upstream regression tests remain enabled;
the application wrapper follows the normal repository lint rules.
