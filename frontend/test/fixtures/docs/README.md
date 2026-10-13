page.html was captured directly from the Go docsPage component on 2026-10-08 by
a temporary package test, removed after capture. DOM tests use the generated
OpenAPI document for full operation coverage and a small synthetic document for
controlled requests, duplicated tags, lazy fields, entity lookup and streams.
Requests in these tests are mocked; no production mutation or financial action
ran. Browser/phone checks and Try it against the local fixture remain separate.
The test mount removes the external module loader because Vitest imports these
same controllers directly and happy-dom disables external JavaScript loading.
