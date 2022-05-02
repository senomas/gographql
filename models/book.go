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

type Book struct {
	ID      int
	Title   string
	Author  Author
	Reviews []Review
}

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
			},
		},
	},
)

func BookQueries(db *sql.DB, fields graphql.Fields) graphql.Fields {
	fields["book"] = &graphql.Field{
		Type:        BookType,
		Description: "book by ID",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Resolve: BookResolver(db),
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
		Resolve: BooksResolver(db),
	}
	return fields
}

func BookMutations(db *sql.DB, fields graphql.Fields) graphql.Fields {
	fields["createBook"] = &graphql.Field{
		Type:        BookType,
		Description: "create new book",
		Args: graphql.FieldConfigArgument{
			"title": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"author": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: CreateBookResolver(db),
	}
	return fields
}

func BookResolver(db *sql.DB) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		var query string
		var fields []string
		if q, fs, err := GenerateBookQuery(p.Info.FieldASTs[0], false); err != nil {
			return nil, err
		} else {
			q.WriteString(" WHERE b.id = $1")
			query = q.String()
			fields = fs
		}

		var reviewQuery string
		var reviewFields []string
		if q, fs, err := GenerateReviewQuery(p.Info.FieldASTs[0]); err != nil {
			return nil, err
		} else {
			q.WriteString(" WHERE r.book_id = $1")
			reviewQuery = q.String()
			reviewFields = fs
		}

		var params []interface{}
		if id, ok := p.Args["id"].(int); ok {
			params = append(params, id)
		} else {
			return nil, errors.New("parameter id required")
		}
		rows, err := db.Query(query, params...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		if rows.Next() {
			var book Book
			err := rows.Scan(BookPointer(fields, &book)...)
			if err != nil {
				return nil, err
			}

			if len(reviewFields) > 0 {
				rows, err := db.Query(reviewQuery, params...)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				var review Review
				preview := ReviewPointer(reviewFields, &review)
				for rows.Next() {
					err := rows.Scan(preview...)
					if err != nil {
						return nil, err
					}
					book.Reviews = append(book.Reviews, review)
				}
			}
			return book, nil
		}
		return nil, nil
	}
}

func BooksResolver(db *sql.DB) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		var params []interface{}
		where := []string{}
		if v, ok := p.Args["id"]; ok {
			params = append(params, v)
			where = append(where, fmt.Sprintf("b.id = $%v", len(params)))
		}
		if v, ok := p.Args["title"]; ok {
			params = append(params, v)
			where = append(where, fmt.Sprintf("b.title = $%v", len(params)))
		}
		if v, ok := p.Args["title_like"]; ok {
			params = append(params, v)
			where = append(where, fmt.Sprintf("b.title LIKE $%v", len(params)))
		}
		joinAuthor := false
		if v, ok := p.Args["author"]; ok {
			joinAuthor = true
			params = append(params, v)
			where = append(where, fmt.Sprintf("a.name = $%v", len(params)))
		}
		if v, ok := p.Args["author_like"]; ok {
			joinAuthor = true
			params = append(params, v)
			where = append(where, fmt.Sprintf("a.name LIKE $%v", len(params)))
		}

		var query string
		var fields []string
		if q, fs, err := GenerateBookQuery(p.Info.FieldASTs[0], joinAuthor); err != nil {
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

		var reviewQuery string
		var reviewFields []string
		if q, fs, err := GenerateReviewQuery(p.Info.FieldASTs[0]); err != nil {
			return nil, err
		} else {
			q.WriteString(" WHERE r.book_id = ANY($1)")
			reviewQuery = q.String()
			reviewFields = fs
		}

		var rows *sql.Rows
		if v, err := db.Query(query, params...); err != nil {
			return nil, err
		} else {
			rows = v
		}
		defer rows.Close()

		var books []*Book
		var book Book
		pbook := BookPointer(fields, &book)
		for rows.Next() {
			err := rows.Scan(pbook...)
			if err != nil {
				return nil, err
			}
			b := book
			books = append(books, &b)
		}

		if len(reviewFields) > 0 {
			bookMap := make(map[int]*Book)

			var ids []int
			for _, book := range books {
				ids = append(ids, book.ID)
				bookMap[book.ID] = book
			}

			rows, err := db.Query(reviewQuery, pq.Array(ids))
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var review Review
			preview := ReviewPointer(reviewFields, &review)
			for rows.Next() {
				err := rows.Scan(preview...)
				if err != nil {
					return nil, err
				}
				if book, ok := bookMap[review.BookID]; ok {
					r := review
					book.Reviews = append(book.Reviews, r)
				}
			}
		}
		return books, nil
	}
}

func CreateBookResolver(db *sql.DB) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		var params []interface{}
		if v, ok := p.Args["author"]; ok {
			params = append(params, v)
		} else {
			return nil, errors.New("parameter author required")
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

					p.Args["id"] = id
					return BookResolver(db)(p)
				}
			}
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
