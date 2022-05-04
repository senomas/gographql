package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/graph-gophers/dataloader"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

type Book struct {
	ID      int
	Title   string
	Author  Author
	Reviews []Review
}

func (l *Loader) getBooks(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ps := make([]*Query, len(keys))
	results := make([]*dataloader.Result, len(keys))
	for ix, key := range keys {
		raw := key.Raw()
		if v, ok := raw.(graphql.ResolveParams); ok {
			ps[ix] = generateBookQuery(v)
			results[ix] = &dataloader.Result{}
		} else {
			results[ix] = &dataloader.Result{Error: fmt.Errorf("invalid %#v", raw)}
		}
	}

	queries := make(map[string]*[]int)
	for ix, p := range ps {
		query := p.Query()
		if qa, ok := queries[query]; ok {
			*qa = append(*qa, ix)
		} else {
			queries[query] = &[]int{ix}
		}
	}

	for qstr, qa := range queries {
		var rows *sql.Rows
		query := ps[(*qa)[0]]
		if len(*qa) == 1 {
			if v, err := l.conn.Query(qstr, query.Params...); err == nil {
				rows = v
			} else {
				results[(*qa)[0]].Error = err
			}
			fmt.Println("STEP 4")
		} else {
			if query.QueryWhere() == "b.id = $1" {
				query.Where[0] = "b.id = ANY($1)"
				qstr = query.Query()
			}
			if v, err := l.conn.Query(qstr, query.Params...); err == nil {
				rows = v
			} else {
				results[(*qa)[0]].Error = err
			}
		}
		if rows != nil {
			var book Book
			var books []*Book
			pbook := BookPointer(query.Fields, &book)
			for rows.Next() {
				if err := rows.Scan(pbook...); err != nil {
					for _, qi := range *qa {
						results[qi].Error = err
					}
					break
				}
				b := book
				books = append(books, &b)
			}
			for _, qi := range *qa {
				results[qi].Data = books
			}
		}
	}
	return results
}

func (l *Loader) LoadBooks(ctx context.Context, p graphql.ResolveParams) (interface{}, error) {
	thunk := l.booksLoader.Load(ctx, NewDataKey(p))
	if res, err := thunk(); res != nil {
		if p.Info.FieldName == "books" {
			return res.([]*Book), err
		}
		return res.([]*Book)[0], err
	} else {
		return nil, err
	}
}

type Query struct {
	Fields []string
	Froms  []string
	Where  []string
	Params []interface{}
}

func generateBookQuery(p graphql.ResolveParams) *Query {
	q := Query{Fields: []string{"b.id"}, Froms: []string{"books b"}}

	for _, s := range p.Info.FieldASTs[0].SelectionSet.Selections {
		cf := s.(*ast.Field)
		if cf.SelectionSet != nil {
			// skip
		} else if cf.Name.Value != "id" {
			q.Fields = append(q.Fields, fmt.Sprintf("b.%s", cf.Name.Value))
		}
	}

	if v, ok := p.Args["id"]; ok {
		q.Params = append(q.Params, v)
		q.Where = append(q.Where, fmt.Sprintf("b.id = $%v", len(q.Params)))
	}

	return &q
}

func (q *Query) Query() string {
	sb := strings.Builder{}
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(q.Fields, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(strings.Join(q.Froms, ", "))
	if len(q.Where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(q.Where, " AND "))
	}

	return sb.String()
}

func (q *Query) QueryWhere() string {
	return strings.Join(q.Where, " AND ")
}

func BookPointer(fields []string, book *Book) []interface{} {
	pointer := make([]interface{}, len(fields))
	for i, f := range fields {
		switch f {
		case "b.id":
			pointer[i] = &book.ID
		case "b.title":
			pointer[i] = &book.Title
		case "a.id":
			pointer[i] = &book.Author.ID
		case "a.name":
			pointer[i] = &book.Author.Name
		}
	}
	return pointer
}
