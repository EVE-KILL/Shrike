# Public kill-feed components

These files are normal Go-rendered routes/fragments through `coldTopReaderStub`:
`/`, `/kills/big`, page 2 at limit 10, valuable ships and the corporations top box.
The temporary capture test was removed. No database data was used. Tests mount
these actual Go component scopes/actions and vary response attributes to exercise
paging, races, retries and local relay arrivals.
