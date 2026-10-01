package brightspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// listPage covers both page shapes Brightspace uses: an ObjectListPage
// (Next link, Objects) and a PagedResultSet (PagingInfo bookmark, Items).
type listPage[T any] struct {
	Next       *string `json:"Next"`
	Objects    []T     `json:"Objects"`
	PagingInfo struct {
		Bookmark     string `json:"Bookmark"`
		HasMoreItems bool   `json:"HasMoreItems"`
	} `json:"PagingInfo"`
	Items []T `json:"Items"`
}

// getAll reads every page of a paged listing (up to maxPages) and returns
// all items. path has no query string; q holds the query parameters.
func getAll[T any](ctx context.Context, c *Client, path string, q url.Values) ([]T, error) {
	var all []T
	next := path
	if len(q) > 0 {
		next += "?" + q.Encode()
	}
	for range maxPages {
		var raw json.RawMessage
		if err := c.get(ctx, next, &raw); err != nil {
			return nil, err
		}
		// Some routes answer with a bare array and no paging information.
		if t := bytes.TrimSpace(raw); len(t) > 0 && t[0] == '[' {
			var items []T
			if err := json.Unmarshal(t, &items); err != nil {
				return nil, fmt.Errorf("GET %s: decode response: %w", path, err)
			}
			return append(all, items...), nil
		}
		var page listPage[T]
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, fmt.Errorf("GET %s: decode response: %w", path, err)
		}
		all = append(all, page.Objects...)
		all = append(all, page.Items...)

		switch {
		case page.Next != nil && *page.Next != "":
			// The session goes to the instance only: never follow a link
			// to another host.
			u, err := url.Parse(*page.Next)
			if err != nil {
				return nil, fmt.Errorf("GET %s: bad next link: %w", path, err)
			}
			if u.IsAbs() && u.Scheme+"://"+u.Host != c.baseURL {
				return nil, fmt.Errorf("GET %s: next link leaves the Brightspace instance", path)
			}
			next = u.RequestURI()
		case page.PagingInfo.HasMoreItems && page.PagingInfo.Bookmark != "":
			nq := url.Values{}
			for k, v := range q {
				nq[k] = v
			}
			nq.Set("bookmark", page.PagingInfo.Bookmark)
			next = path + "?" + nq.Encode()
		default:
			return all, nil
		}
	}
	return all, nil
}
