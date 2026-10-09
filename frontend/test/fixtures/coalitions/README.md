# Coalition component captures

These files were rendered by `internal/frontend` using `testCoalitionReader` from
`coalition_page_test.go`, through the ordinary directory, detail and fragment HTTP
routes. The temporary capture test was removed after generating them. No database,
production records or job dispatch were used. Browser controller tests strip the
shell scripts and mount the component with its Go-declared Stimulus actions.

`directory.html` intentionally contains pending activity totals; `activity.html`
is their independent server fragment. `detail-fragment.html` provides the revision,
members and history used by controlled refresh/saving tests.
