package panel

import (
	"context"
	"fmt"
	"net/http"
)

const pageSize = 50

// page is one page of a Marzneshin fastapi-pagination response.
type page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
}

// listAll walks every page of a paginated collection at path and returns all items.
func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	var all []T
	for pageNum := 1; ; pageNum++ {
		var body page[T]
		query := fmt.Sprintf("%s?page=%d&size=%d", path, pageNum, pageSize)
		if err := c.do(ctx, http.MethodGet, query, nil, &body); err != nil {
			return nil, err
		}
		all = append(all, body.Items...)
		if len(body.Items) == 0 || pageNum >= body.Pages {
			return all, nil
		}
	}
}
