Captured directly from Go mcpPage/mcpOutput on 2026-10-08 using the registered
catalog and a fixed schema/error view. The temporary capture test was removed.
DOM mounts remove the external module and stylesheet loaders because Vitest
imports the same controller and the test checks DOM behavior without network
asset loading. Real styling is a separate browser gate. Tests cover
filtering, lazy/cached schema reads, failures/retry, cancellation, page return
and clipboard feedback. Browser/phone checks remain pending; no MCP tool was run
against production and no production data was changed.
