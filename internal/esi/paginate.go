package esi

import (
	"context"
	"fmt"
	"strings"
)

// pageSize is what ESI returns per page on the paginated endpoints. A short page
// means the last page, which is the only end-of-list signal those endpoints
// give.
const pageSize = 1000

// maxPages bounds a walk. No legitimate list runs to a thousand pages; hitting
// this means ESI is repeating itself and the loop would otherwise never end.
const maxPages = 1000

// GetAllPages walks a paginated endpoint to the end and returns the whole list.
//
// Partial results are returned alongside a failure rather than discarded: for a
// killmail backfill, nine pages of history is worth keeping even when the tenth
// times out, and the caller can retry from a bookmark.
func GetAllPages[T any](ctx context.Context, c *Client, path string, accessToken string) ([]T, Response[[]T], error) {
	var all []T
	last := Response[[]T]{Status: 200}

	for page := 1; page <= maxPages; page++ {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		paged := fmt.Sprintf("%s%spage=%d", path, sep, page)

		var res Response[[]T]
		var err error
		if accessToken == "" {
			res, err = Get[[]T](ctx, c, paged)
		} else {
			res, err = GetAuthenticated[[]T](ctx, c, paged, accessToken)
		}
		if err != nil {
			return all, res, err
		}
		last = res

		if !res.OK() {
			return all, res, nil
		}

		all = append(all, *res.Data...)
		if len(*res.Data) < pageSize {
			break
		}
	}

	last.Data = &all
	return all, last, nil
}
