package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
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

func ReviewMutations(db *sql.DB, fields graphql.Fields) graphql.Fields {
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
		Resolve: CreateReviewResolver(db),
	}
	return fields
}

func CreateReviewResolver(db *sql.DB) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
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
}

func GenerateReviewQuery(f ast.Selection) (*strings.Builder, []string, error) {
	var froms = []string{"reviews r"}
	var fields []string
	for _, s := range f.GetSelectionSet().Selections {
		cf := s.(*ast.Field)
		if cf.SelectionSet != nil {
			if cf.Name.Value == "author" {
			} else if cf.Name.Value == "reviews" {
				fields = append(fields, "r.book_id")
				for _, s := range cf.SelectionSet.Selections {
					cf := s.(*ast.Field)
					if cf.SelectionSet != nil {
						return nil, nil, fmt.Errorf("unknown field '%s' in Author", cf.Name.Value)
					} else if cf.Name.Value != "book_id" {
						fields = append(fields, fmt.Sprintf("r.%s", cf.Name.Value))
					}
				}
			} else {
				return nil, nil, fmt.Errorf("unknown field '%s' in Book", cf.Name.Value)
			}
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