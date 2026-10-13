The status component and snapshot were captured on 2026-10-08 by a temporary
internal/frontend package test, removed after capture. page.html comes directly
from statusPage(). snapshot.json was marshalled from the actual internal/status
Status/SystemInfo/DatabaseInfo/ESI/Coverage/Token types and zkb.IngestStats using
fixed test values. Optional external sources remain null as in the Go contract.

DOM tests use the actual shared relay transport with a controlled WebSocket,
covering reconnect/backoff, visibility, late messages, bfcache and tooltips. Extra
external-panel/malformed-data cases are synthetic. Live browser/mobile and real
production status delivery still need validation; no production data was changed.
