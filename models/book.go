package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/senomas/gographql/data"
)

var BookType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Book",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"title": &graphql.Field{
				Type: graphql.String,
			},
			"author": &graphql.Field{
				Type: AuthorType,
			},
			"reviews": &graphql.Field{
				Type: graphql.NewList(ReviewType),
				Args: graphql.FieldConfigArgument{
					"query_limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"query_offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: ReviewsResolver,
			},
		},
	},
)

func BookQueries(fields graphql.Fields) graphql.Fields {
	fields["book"] = &graphql.Field{
		Type:        BookType,
		Description: "book by ID",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Resolve: BookResolver,
	}
	fields["books"] = &graphql.Field{
		Type:        graphql.NewList(BookType),
		Description: "Get list of book",
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
			"title": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"title_like": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"author": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"author_like": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: BookResolver,
	}
	return fields
}

func BookMutations(fields graphql.Fields) graphql.Fields {
	fields["createBook"] = &graphql.Field{
		Type:        BookType,
		Description: "create new book",
		Args: graphql.FieldConfigArgument{
			"title": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"author_id": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"author": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: CreateBookResolver,
	}
	return fields
}

func BookResolver(p graphql.ResolveParams) (interface{}, error) {
	type result struct {
		data interface{}
		err  error
	}
	ch := make(chan *result, 1)
	go func() {
		defer close(ch)
		loader := p.Context.Value(ContextKeyLoader).(*data.Loader)

		data, err := loader.LoadBooks(p.Context, p)
		ch <- &result{data: data, err: err}
	}()
	return func() (interface{}, error) {
		r := <-ch
		return r.data, r.err
	}, nil
}

func CreateBookResolver(p graphql.ResolveParams) (interface{}, error) {
	db := p.Context.Value(ContextKey("db")).(*sql.DB)
	var params []interface{}
	if v, ok := p.Args["author_id"]; ok {
		params = append(params, v)
	} else {
		if v, ok := p.Args["author"]; ok {
			params = append(params, v)
		} else {
			return nil, errors.New("parameter author or author_id required")
		}

		if rows, err := db.Query("SELECT id FROM authors WHERE name = $1", params...); err != nil {
			return nil, fmt.Errorf("failed to create author, err %v", err)
		} else {
			{
				defer rows.Close()
				if rows.Next() {
					rows.Scan(&params[0])
				} else {
					return nil, fmt.Errorf("author '%s' not found", params[0])
				}
			}
		}
	}
	if v, ok := p.Args["title"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter title required")
	}
	if tx, err := db.Begin(); err != nil {
		return nil, fmt.Errorf("failed begin tx, err %v", err)
	} else {
		if rows, err := tx.Query("INSERT INTO books (author_id, title) VALUES ($1, $2) RETURNING id", params...); err != nil {
			return nil, fmt.Errorf("failed to create book, err %v", err)
		} else {
			var id int
			{
				defer rows.Close()
				if rows.Next() {
					rows.Scan(&id)
				} else {
					return nil, errors.New("failed to insert authors")
				}
				tx.Commit()
			}

			p.Args = make(map[string]interface{})
			p.Args["id"] = id
			return BookResolver(p)
		}
	}
}

func GenerateBookQuery(f ast.Selection, joinAuthor bool) (*strings.Builder, []string, error) {
	var froms = []string{"books b"}
	var fields = []string{"b.id"}
	if joinAuthor {
		froms = append(append([]string{"("}, froms...), " LEFT JOIN authors a ON b.author_id = a.id)")
	}
	for _, s := range f.GetSelectionSet().Selections {
		cf := s.(*ast.Field)
		if cf.SelectionSet != nil {
			if cf.Name.Value == "author" {
				if !joinAuthor {
					froms = append(append([]string{"("}, froms...), " LEFT JOIN authors a ON b.author_id = a.id)")
				}
				for _, s := range cf.SelectionSet.Selections {
					cf := s.(*ast.Field)
					if cf.SelectionSet != nil {
						return nil, nil, fmt.Errorf("unknown field '%s' in Author", cf.Name.Value)
					} else {
						fields = append(fields, fmt.Sprintf("a.%s", cf.Name.Value))
					}
				}
			} else if cf.Name.Value == "reviews" {
			} else {
				return nil, nil, fmt.Errorf("unknown field '%s' in Book", cf.Name.Value)
			}
		} else {
			if cf.Name.Value != "id" {
				fields = append(fields, fmt.Sprintf("b.%s", cf.Name.Value))
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

func BookPointer(fields []string, book *data.Book) []interface{} {
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
