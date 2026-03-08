package linear

import "context"

// page represents a single page of results with pagination metadata.
type page[T any] struct {
	Nodes    []T      `json:"nodes"`
	PageInfo pageInfo `json:"pageInfo"`
}

// pageExtractor extracts a page from a response data struct.
type pageExtractor[D any, T any] func(data *D) page[T]

// fetchAll fetches all pages of a paginated query, accumulating nodes.
func fetchAll[D any, T any](
	ctx context.Context,
	c *LinearClient,
	query string,
	vars map[string]any,
	extract pageExtractor[D, T],
) ([]T, error) {
	var all []T
	var cursor string

	for {
		// Set pagination variables.
		v := make(map[string]any, len(vars)+2)
		for k, val := range vars {
			v[k] = val
		}
		v["first"] = defaultPageSize
		if cursor != "" {
			v["after"] = cursor
		}

		var data D
		if err := c.do(ctx, query, v, &data); err != nil {
			return nil, err
		}

		p := extract(&data)
		all = append(all, p.Nodes...)

		if !p.PageInfo.HasNextPage {
			break
		}
		cursor = p.PageInfo.EndCursor
	}

	return all, nil
}
