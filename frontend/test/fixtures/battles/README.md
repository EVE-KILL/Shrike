These are captured Go/templ components from the disposable browser facade on
2026-10-08, without the surrounding site shell. The populated composition/replay/
detail snapshots use public battle 1172169. The generator preview uses only the
named local fixture systems/participants seeded by internal/testfixtures. No
production report was saved. Tests explicitly vary state, rows and responses.

Keep raw template HTML intact. happy-dom loses table rows when parsing a template
inside a section; the controller tests restore each template from the captured
raw HTML before connecting Stimulus. Browser checks use the actual renderer.
