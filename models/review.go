package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

var ReviewType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Review",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"book_id": &graphql.Field{
				Type: graphql.Int,
			},
			"star": &graphql.Field{
				Type: graphql.Int,
			},
			"body": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

func ReviewMutations(fields graphql.Fields) graphql.Fields {
	fields["createReview"] = &graphql.Field{
		Type:        ReviewType,
		Description: "create new review",
		Args: graphql.FieldConfigArgument{
			"book_id": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"star": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"body": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: CreateReviewResolver,
	}
	return fields
}

func ReviewsResolver(p graphql.ResolveParams) (interface{}, error) {
	db := p.Context.Value(ContextKeyDB).(*sql.DB)
	cache := p.Context.Value(ContextKeyCache).(map[string]interface{})

	var query string
	var fields []string
	var params []interface{}
	where := []string{}

	var book *Book
	var books *[]Book

	if p.Source != nil {
		switch v := p.Source.(type) {
		case Book:
			book = &v
		case *Book:
			book = v
		default:
			return nil, fmt.Errorf("unexpected type %#v", v)
		}

		if v, ok := cache["reviews"]; ok {
			reviewMap := *v.(*map[int]*[]Review)
			reviews := reviewMap[book.ID]
			return *reviews, nil
		}

		if v, ok := cache["books"]; ok {
			books = v.(*[]Book)

			var ids []int
			for _, b := range *books {
				ids = append(ids, b.ID)
			}
			params = append(params, pq.Array(ids))
			where = append(where, fmt.Sprintf(" r.book_id = ANY($%v)", len(params)))
		} else {
			params = append(params, book.ID)
			where = append(where, fmt.Sprintf(" r.book_id = $%v", len(params)))
		}
	}

	if q, fs, err := GenerateReviewQuery(p.Info.FieldASTs[0]); err != nil {
		return nil, err
	} else {
		if len(where) > 0 {
			q.WriteString(" WHERE ")
			for i, w := range where {
				if i > 0 {
					q.WriteString(" AND ")
				}
				q.WriteString(w)
			}
		}
		if v, ok := p.Args["query_limit"]; ok {
			q.WriteString(fmt.Sprintf(" LIMIT %v", v))
		}
		if v, ok := p.Args["query_offset"]; ok {
			q.WriteString(fmt.Sprintf(" OFFSET %v", v))
		}
		query = q.String()
		fields = fs
	}

	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var review Review
	preview := ReviewPointer(fields, &review)

	if book != nil && books != nil {
		var reviews []Review
		reviewMap := make(map[int]*[]Review)
		cache["reviews"] = &reviewMap

		for rows.Next() {
			err := rows.Scan(preview...)
			if err != nil {
				return nil, err
			}
			if review.BookID == book.ID {
				reviews = append(reviews, review)
			} else {
				if revs, ok := reviewMap[review.BookID]; ok {
					*revs = append(*revs, review)
				} else {
					v := []Review{review}
					reviewMap[review.BookID] = &v
				}
			}
		}
		return reviews, nil
	} else {
		var reviews []Review
		for rows.Next() {
			err := rows.Scan(preview...)
			if err != nil {
				return nil, err
			}
			reviews = append(reviews, review)
		}
		return reviews, nil
	}
}

func CreateReviewResolver(p graphql.ResolveParams) (interface{}, error) {
	db := p.Context.Value(ContextKeyDB).(*sql.DB)
	var params []interface{}
	if v, ok := p.Args["book_id"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter book_id required")
	}
	if v, ok := p.Args["star"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter star required")
	}
	if v, ok := p.Args["body"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter body required")
	}
	if tx, err := db.Begin(); err != nil {
		return nil, fmt.Errorf("failed begin tx, err %v", err)
	} else {
		if rows, err := tx.Query("INSERT INTO reviews (book_id, star, body) VALUES ($1, $2, $3) RETURNING id", params...); err != nil {
			return nil, fmt.Errorf("failed to create review, err %v", err)
		} else {
			var id int
			{
				defer rows.Close()
				if rows.Next() {
					rows.Scan(&id)
				} else {
					return nil, errors.New("failed to insert review")
				}
				tx.Commit()
			}

			if rows, err := db.Query("SELECT id, book_id, star, body FROM reviews WHERE id = $1", id); err != nil {
				return nil, err
			} else {
				if rows.Next() {
					var review Review
					rows.Scan(&review.ID, &review.BookID, &review.Star, &review.Body)

					return review, nil
				}
			}
		}
	}
	return nil, nil
}

func GenerateReviewQuery(f ast.Selection) (*strings.Builder, []string, error) {
	var froms = []string{"reviews r"}
	fields := []string{"r.book_id"}
	for _, s := range f.GetSelectionSet().Selections {
		cf := s.(*ast.Field)
		if cf.SelectionSet != nil {
			return nil, nil, fmt.Errorf("unknown field '%s' in Review", cf.Name.Value)
		} else if cf.Name.Value != "book_id" {
			fields = append(fields, fmt.Sprintf("r.%s", cf.Name.Value))
		}
	}
	query := strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(fields, ", "))
	query.WriteString(" FROM ")
	query.WriteString(strings.Join(froms, ""))

	return &query, fields, nil
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
