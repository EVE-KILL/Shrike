Captured Go/templ components from the named disposable local PostgreSQL fixture
on 2026-10-08. Report a462a5e9c726c1e7e210d9b8452bc3a9 uses the existing fixture
pilots and killmails from 2026-09-24. No production report was created or updated.

Keep raw template HTML intact. happy-dom loses table rows inside templates when
parsing their enclosing component; the controller tests restore those templates
from this raw HTML before Stimulus connects. DOM and transport regressions do not
replace desktop/mobile browser validation.
