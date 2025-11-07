package pagination

import (
	"context"
	"reflect"
)

// PageIterator provides automatic pagination for list endpoints
type PageIterator[T any] struct {
	items   []T
	idx     int
	page    *Page[T]
	err     error
	fetcher func(ctx context.Context, params interface{}) (*Page[T], error)
	params  interface{}
	ctx     context.Context
}

// Page represents a paginated response
type Page[T any] struct {
	Data       []T
	Pagination Pagination
	Fetcher    func(ctx context.Context, params interface{}) (*Page[T], error)
	Params     interface{}
	Ctx        context.Context
}

// Pagination represents pagination information
type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	Count   int  `json:"count"`
	HasMore bool `json:"hasMore"`
}

// NewPageIterator creates a new page iterator
func NewPageIterator[T any](
	ctx context.Context,
	params interface{},
	fetcher func(ctx context.Context, params interface{}) (*Page[T], error),
) *PageIterator[T] {
	return &PageIterator[T]{
		ctx:     ctx,
		params:  params,
		fetcher: fetcher,
		idx:     -1,
	}
}

// Next advances the iterator to the next item
func (iter *PageIterator[T]) Next() bool {
	// Move to next item in current page
	iter.idx++
	if iter.idx < len(iter.items) {
		return true
	}

	// Need to fetch next page
	if iter.page == nil {
		// First page
		page, err := iter.fetcher(iter.ctx, iter.params)
		if err != nil {
			iter.err = err
			return false
		}
		iter.page = page
		iter.items = page.Data
		iter.idx = 0
		return len(iter.items) > 0
	}

	// Check if there are more pages
	if !iter.page.Pagination.HasMore {
		return false
	}

	// Fetch next page
	nextPage, err := iter.page.GetNextPage()
	if err != nil {
		iter.err = err
		return false
	}

	if nextPage == nil {
		return false
	}

	iter.page = nextPage
	iter.items = nextPage.Data
	iter.idx = 0
	return len(iter.items) > 0
}

// Current returns the current item
func (iter *PageIterator[T]) Current() T {
	if iter.idx >= 0 && iter.idx < len(iter.items) {
		return iter.items[iter.idx]
	}
	var zero T
	return zero
}

// Err returns any error encountered during iteration
func (iter *PageIterator[T]) Err() error {
	return iter.err
}

// GetNextPage fetches the next page of results
func (p *Page[T]) GetNextPage() (*Page[T], error) {
	if !p.Pagination.HasMore {
		return nil, nil
	}

	// Update params with next offset using reflection
	newOffset := p.Pagination.Offset + p.Pagination.Count

	// Use reflection to set Offset field
	paramsValue := reflect.ValueOf(p.Params)
	if paramsValue.Kind() == reflect.Pointer {
		paramsValue = paramsValue.Elem()
	}

	// Create a copy of the params
	newParamsValue := reflect.New(paramsValue.Type()).Elem()
	newParamsValue.Set(paramsValue)

	// Set the Offset field
	offsetField := newParamsValue.FieldByName("Offset")
	if offsetField.IsValid() && offsetField.CanSet() {
		if offsetField.Kind() == reflect.Pointer {
			offsetPtr := reflect.New(offsetField.Type().Elem())
			offsetPtr.Elem().SetInt(int64(newOffset))
			offsetField.Set(offsetPtr)
		} else {
			offsetField.SetInt(int64(newOffset))
		}
	}

	// Get pointer to new params
	newParams := newParamsValue.Addr().Interface()
	return p.Fetcher(p.Ctx, newParams)
}

// HasNextPage returns true if there are more pages available
func (p *Page[T]) HasNextPage() bool {
	return p.Pagination.HasMore
}
