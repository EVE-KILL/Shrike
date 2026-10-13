Captured through the normal Go market/item handlers on 2026-10-08 using the
existing testUniverseReader. Browser/standalone pages were reduced to their
data-market-page roots without changing markup; item/filtered panels and tree
fragments retain original classes and component markers. The temporary capture
test was removed. Unit mounts omit external module/stylesheet loaders, since
Vitest imports the same controllers and layout requires a separate browser gate.
No production market data or database was changed.
