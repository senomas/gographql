package data

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/graph-gophers/dataloader"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/lib/pq"
)

type Review struct {
	ID     int
	Star   int
	Body   string
	BookID int `json:"book_id"`
}

func (l *Loader) getReviews(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	log.Printf("BATCH REVIEW QUERY -- %v\n", len(keys))
	ps := make([]*Query, len(keys))
	results := make([]*dataloader.Result, len(keys))
	for ix, key := range keys {
		raw := key.Raw()
		if v, ok := raw.(graphql.ResolveParams); ok {
			ps[ix] = generateReviewQuery(v)
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
		fmt.Printf("BATCH REVIEW QUERY 1 '%v' %#v\n", qstr, qa)
		query := ps[(*qa)[0]]
		fmt.Printf("BATCH REVIEW QUERY 2 %#v\n", query)
		if len(*qa) == 1 {
			fmt.Println("STEP REVIEW 1")
			if rows, err := l.conn.Query(qstr, query.Params...); err == nil {
				fmt.Println("STEP REVIEW 2")
				if data, err := readReviewRows(query, rows); err != nil {
					for _, qi := range *qa {
						results[qi].Error = err
					}
				} else {
					for _, qi := range *qa {
						results[qi].Data = data
					}
				}
			} else {
				fmt.Printf("STEP REVIEW 3 %v\n", err)
				results[(*qa)[0]].Error = err
			}
		} else {
			if query.QueryWhere() == "r.book_id = $1" {
				query.Where[0] = "r.book_id = ANY($1)"
				qstr = query.Query()
				var ids []int
				for _, qi := range *qa {
					ids = append(ids, ps[qi].Params[0].(int))
				}
				fmt.Printf("BATCH REVIEW QUERY 3 '%v' %#v\n", qstr, ids)
				if rows, err := l.conn.Query(qstr, pq.Array(ids)); err == nil {
					if data, err := readReviewRows(query, rows); err != nil {
						for _, qi := range *qa {
							results[qi].Error = err
						}
					} else {
						for _, qi := range *qa {
							var result []*Review
							bookID := ps[qi].Params[0].(int)
							for _, r := range data {
								if r.BookID == bookID {
									result = append(result, r)
								}
							}
							results[qi].Data = result
						}
					}
				} else {
					for _, qi := range *qa {
						results[qi].Error = err
					}
				}
			} else {
				fmt.Printf("BATCH REVIEW QUERY 4 '%v' %#v\n", qstr, qa)
				for _, qi := range *qa {
					if rows, err := l.conn.Query(qstr, ps[qi].Params...); err == nil {
						if data, err := readReviewRows(query, rows); err != nil {
							results[qi].Error = err
						} else {
							results[qi].Data = data
						}
					} else {
						results[qi].Error = err
					}
				}
			}
		}
	}
	fmt.Println("DONE BATCH REVIEW QUERY")
	return results
}

func readReviewRows(query *Query, rows *sql.Rows) ([]*Review, error) {
	var review Review
	var reviews []*Review
	pbook := ReviewPointer(query.Fields, &review)
	for rows.Next() {
		if err := rows.Scan(pbook...); err != nil {
			return nil, err
			break
		}
		r := review
		reviews = append(reviews, &r)
	}
	return reviews, nil
}

func (l *Loader) LoadReviews(ctx context.Context, p graphql.ResolveParams) (interface{}, error) {
	thunk := l.reviewsLoader.Load(ctx, NewDataKey(p))
	if res, err := thunk(); res != nil {
		if p.Info.FieldName == "reviews" {
			return res.([]*Review), err
		}
		return res.([]*Review)[0], err
	} else {
		return nil, err
	}
}

func generateReviewQuery(p graphql.ResolveParams) *Query {
	q := Query{Fields: []string{"r.id", "r.book_id"}, Froms: []string{"reviews r"}}

	for _, s := range p.Info.FieldASTs[0].SelectionSet.Selections {
		cf := s.(*ast.Field)
		if cf.SelectionSet != nil {
			// skip
		} else if cf.Name.Value != "id" {
			q.Fields = append(q.Fields, fmt.Sprintf("r.%s", cf.Name.Value))
		}
	}

	if p.Source != nil {
		fmt.Printf("SOURCE %#v\n", p.Source)
		q.Params = append(q.Params, p.Source.(*Book).ID)
		q.Where = append(q.Where, fmt.Sprintf("r.book_id = $%v", len(q.Params)))
	} else if v, ok := p.Args["id"]; ok {
		q.Params = append(q.Params, v)
		q.Where = append(q.Where, fmt.Sprintf("r.id = $%v", len(q.Params)))
	}

	return &q
}

func ReviewPointer(fields []string, review *Review) []interface{} {
	pointer := make([]interface{}, len(fields))
	for i, f := range fields {
		switch f {
		case "r.id":
			pointer[i] = &review.ID
		case "r.star":
			pointer[i] = &review.Star
		case "r.body":
			pointer[i] = &review.Body
		case "r.book_id":
			pointer[i] = &review.BookID
		}
	}
	return pointer
}
