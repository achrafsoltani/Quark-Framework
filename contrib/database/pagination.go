package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Page represents a page of results.
type Page[T any] struct {
	Items      []T  `json:"items"`
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

// PaginationParams holds pagination parameters.
type PaginationParams struct {
	Page    int
	PerPage int
	Offset  int
}

// NewPaginationParams creates pagination params with defaults.
func NewPaginationParams(page, perPage, defaultPerPage, maxPerPage int) PaginationParams {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

// QueryBuilder helps build SQL queries with pagination.
type QueryBuilder struct {
	baseQuery   string
	countQuery  string
	whereClause string
	orderBy     string
	args        []interface{}
	offset      int
	limit       int
}

// NewQueryBuilder creates a new query builder.
func NewQueryBuilder(baseQuery string) *QueryBuilder {
	return &QueryBuilder{
		baseQuery: baseQuery,
		args:      make([]interface{}, 0),
	}
}

// Where adds a WHERE clause.
func (qb *QueryBuilder) Where(clause string, args ...interface{}) *QueryBuilder {
	if qb.whereClause == "" {
		qb.whereClause = "WHERE " + clause
	} else {
		qb.whereClause += " AND " + clause
	}
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy adds an ORDER BY clause.
func (qb *QueryBuilder) OrderBy(clause string) *QueryBuilder {
	qb.orderBy = "ORDER BY " + clause
	return qb
}

// Paginate adds LIMIT and OFFSET.
func (qb *QueryBuilder) Paginate(p PaginationParams) *QueryBuilder {
	qb.limit = p.PerPage
	qb.offset = p.Offset
	return qb
}

// Build returns the final query and arguments.
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var parts []string
	parts = append(parts, qb.baseQuery)

	if qb.whereClause != "" {
		parts = append(parts, qb.whereClause)
	}
	if qb.orderBy != "" {
		parts = append(parts, qb.orderBy)
	}

	query := strings.Join(parts, " ")
	args := qb.args

	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, qb.limit)
	}
	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, qb.offset)
	}

	return query, args
}

// BuildCount returns a COUNT query.
func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
	// Try to replace SELECT ... FROM with SELECT COUNT(*) FROM
	baseQuery := qb.baseQuery
	selectIdx := strings.Index(strings.ToUpper(baseQuery), "SELECT")
	fromIdx := strings.Index(strings.ToUpper(baseQuery), "FROM")

	var countQuery string
	if selectIdx >= 0 && fromIdx > selectIdx {
		countQuery = "SELECT COUNT(*) " + baseQuery[fromIdx:]
	} else {
		countQuery = "SELECT COUNT(*) FROM (" + baseQuery + ") AS count_query"
	}

	if qb.whereClause != "" {
		countQuery += " " + qb.whereClause
	}

	return countQuery, qb.args
}

// Paginator provides a convenient way to paginate query results.
type Paginator[T any] struct {
	db       Querier
	scanner  func(*sql.Rows) (T, error)
	params   PaginationParams
}

// NewPaginator creates a new paginator.
func NewPaginator[T any](db Querier, scanner func(*sql.Rows) (T, error), params PaginationParams) *Paginator[T] {
	return &Paginator[T]{
		db:      db,
		scanner: scanner,
		params:  params,
	}
}

// Execute runs the paginated query and returns a Page.
func (p *Paginator[T]) Execute(ctx context.Context, qb *QueryBuilder) (*Page[T], error) {
	// Get total count
	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := p.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	// Get items
	qb.Paginate(p.params)
	query, args := qb.Build()

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		item, err := p.scanner(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	totalPages := 0
	if p.params.PerPage > 0 {
		totalPages = (total + p.params.PerPage - 1) / p.params.PerPage
	}

	return &Page[T]{
		Items:      items,
		Page:       p.params.Page,
		PerPage:    p.params.PerPage,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    p.params.Page < totalPages,
	}, nil
}

// PaginateQuery is a convenience function for simple pagination.
func PaginateQuery[T any](
	ctx context.Context,
	db Querier,
	baseQuery string,
	scanner func(*sql.Rows) (T, error),
	params PaginationParams,
	where string,
	args ...interface{},
) (*Page[T], error) {
	qb := NewQueryBuilder(baseQuery)
	if where != "" {
		qb.Where(where, args...)
	}

	paginator := NewPaginator(db, scanner, params)
	return paginator.Execute(ctx, qb)
}

// ScanInto scans a row into a struct using a map of column names to field pointers.
func ScanInto(rows *sql.Rows, dest ...interface{}) error {
	return rows.Scan(dest...)
}
