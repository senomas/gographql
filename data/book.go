package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/graph-gophers/dataloader"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/senomas/gographql/models"
	"gorm.io/gorm"
)

type Book struct {
	ID       int `gorm:"primaryKey"`
	Title    string
	AuthorID int
	Author   Author
	Reviews  []Review
}

func (l *Loader) getBooks(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ps := make([]*Query, len(keys))
	results := make([]*dataloader.Result, len(keys))
	for ix, key := range keys {
		raw := key.Raw()
		if v, ok := raw.(graphql.ResolveParams); ok {
			ps[ix] = generateBookQuery(ctx, v)
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
	return l.booksLoader.Load(ctx, NewDataKey(p))()
}

func generateBookQueryzxxx(ctx context.Context, p graphql.ResolveParams) *gorm.DB {
	db := ctx.Value(models.ContextKeyDB).(*gorm.DB)
	tx := db.Table(p.Info.FieldName)
	var fields []string
	for _, s := range p.Info.FieldASTs[0].SelectionSet.Selections {
		cf := s.(*ast.Field)
		if cf.SelectionSet != nil {
			// skip
		} else if cf.Name.Value != "id" {
			fields = append(fields, cf.Name.Value)
		}
	}
	tx = tx.Select(fields)

	var filters []string
	var params []interface{}
	for k, v := range p.Args {
		filters = append(filters, fmt.Sprintf("%s = ?", k))
		params = append(params, v)
	}

	return tx
}
