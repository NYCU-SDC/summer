package pagination

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
)

type Request struct {
	page   int
	size   int
	sort   string
	sortBy string
}

type Response[T any] struct {
	Items       []T  `json:"items"`
	TotalPages  int  `json:"totalPages"`
	TotalItems  int  `json:"totalItems"`
	CurrentPage int  `json:"currentPage"`
	PageSize    int  `json:"pageSize"`
	HasNextPage bool `json:"hasNextPage"`
}

type Factory[T any] struct {
	maxPageSize     int
	sortableColumns []string
}

func NewFactory[T any](maxPageSize int, sortableColumns []string) Factory[T] {
	return Factory[T]{
		maxPageSize:     maxPageSize,
		sortableColumns: sortableColumns,
	}
}

func (f Factory[T]) GetRequest(r *http.Request) (Request, error) {
	pageParam := r.URL.Query().Get("page")
	sizeParam := r.URL.Query().Get("size")
	sort := r.URL.Query().Get("sort")
	sortBy := r.URL.Query().Get("sortBy")

	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 1 {
		page = 0
	}
	size, err := strconv.Atoi(sizeParam)
	if err != nil || size < 1 {
		size = 10
	}

	if size > f.maxPageSize {
		return Request{}, errors.New("max page size exceeded")
	}
	if !slices.Contains(f.sortableColumns, sort) && sortBy != "" {
		return Request{}, errors.New("invalid sort column")
	}

	return Request{
		page:   page,
		size:   size,
		sort:   sort,
		sortBy: sortBy,
	}, nil
}

func (f Factory[T]) NewResponse(items []T, totalItems int, page, size int) Response[T] {
	totalPages := totalItems / size
	if totalItems%size != 0 {
		totalPages++
	}

	return Response[T]{
		Items:       items,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
		CurrentPage: page,
		PageSize:    size,
		HasNextPage: page < totalPages,
	}
}
