package pagination

import (
	"net/http"
	"slices"
	"strconv"
)

type Request struct {
	Page   int
	Size   int
	Sort   string
	SortBy string
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
	MaxPageSize     int
	SortableColumns []string
}

func NewFactory[T any](maxPageSize int, sortableColumns []string) Factory[T] {
	return Factory[T]{
		MaxPageSize:     maxPageSize,
		SortableColumns: sortableColumns,
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

	if size > f.MaxPageSize {
		return Request{}, ErrInvalidPageOrSize
	}
	if !slices.Contains(f.SortableColumns, sort) && sortBy != "" {
		return Request{}, ErrInvalidSortingField
	}

	return Request{
		Page:   page,
		Size:   size,
		Sort:   sort,
		SortBy: sortBy,
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
		HasNextPage: page+1 < totalPages,
	}
}
