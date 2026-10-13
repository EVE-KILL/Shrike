# Map controller fixture

`page.html` is the normal Go `/map?base=aiid` response, captured without database
or production requests. The temporary capture test was removed. Tests strip
executable scripts and styles, then mount the real feed panel elements. Typed
relay/scope inputs, controlled WebSockets and a fake clock exercise streamed
updates, alarm bands, visibility, freshness, races, row retention and teardown.
