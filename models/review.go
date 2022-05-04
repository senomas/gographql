package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/senomas/gographql/data"
)

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

func ReviewQueries(fields graphql.Fields) graphql.Fields {
	fields["reviews"] = &graphql.Field{
		Type:        graphql.NewList(BookType),
		Description: "Get list of reviews",
		Args: graphql.FieldConfigArgument{
			"query_limit": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"query_offset": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"id": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"star": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Resolve: ReviewsResolver,
	}
	return fields
}

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
	type result struct {
		data interface{}
		err  error
	}
	ch := make(chan *result, 1)
	go func() {
		defer close(ch)
		loader := p.Context.Value(ContextKeyLoader).(*data.Loader)

		data, err := loader.LoadReviews(p.Context, p)
		ch <- &result{data: data, err: err}
	}()
	return func() (interface{}, error) {
		r := <-ch
		return r.data, r.err
	}, nil
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
					var review data.Review
					rows.Scan(&review.ID, &review.BookID, &review.Star, &review.Body)

					return review, nil
				}
			}
		}
	}
	return nil, nil
}
