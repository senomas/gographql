package test_book

import (
	"database/sql/driver"
	"encoding/json"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/graphql-go/graphql"
	"github.com/senomas/gographql/models"
	"github.com/stretchr/testify/assert"
)

func TestBook(t *testing.T) {
	if sqlDB, mock, err := sqlmock.New(); err != nil {
		t.Fatal("init SQLMock Error", err)
	} else {
		defer sqlDB.Close()

		schemaConfig := graphql.SchemaConfig{
			Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: models.CreateFields(sqlDB, models.BookQueries)}),
			Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: models.CreateFields(sqlDB, models.AuthorMutations, models.ReviewMutations, models.BookMutations)}),
		}
		var schema graphql.Schema
		if s, err := graphql.NewSchema(schemaConfig); err != nil {
			log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
		} else {
			schema = s
		}

		t.Run("find books by id", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title FROM books b WHERE b.id = $1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			query := `
			{
				book(id: 2) {
					id
					title
				}
			}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"book": {
			"id": 2,
			"title": "Harry Potter and the Chamber of Secrets"
		}
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find book with author name by id", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE b.id = $1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			query := `
				{
					book(id: 2) {
						id
						title
						author {
							name
						}
					}
				}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"book": {
			"author": {
				"name": "J.K. Rowling"
			},
			"id": 2,
			"title": "Harry Potter and the Chamber of Secrets"
		}
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find book with author name and reviews by id", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE b.id = $1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			mock.ExpectQuery(QuoteMeta(`SELECT r.book_id, r.star, r.body FROM reviews r WHERE r.book_id = $1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "star", "body"}).
				AddRow(2, 5, "The Boy Who Lived").
				AddRow(2, 4, "The stone that must be destroyed"))

			query := `
				{
					book(id: 2) {
						id
						title
						author {
							name
						}
						reviews {
							star
							body
						}
					}
				}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"book": {
			"author": {
				"name": "J.K. Rowling"
			},
			"id": 2,
			"reviews": [
				{
					"body": "The Boy Who Lived",
					"star": 5
				},
				{
					"body": "The stone that must be destroyed",
					"star": 4
				}
			],
			"title": "Harry Potter and the Chamber of Secrets"
		}
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books with author name", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id)`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling").
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling").
				AddRow(3, "Harry Potter and the Prisoner of Azkaban", "J.K. Rowling").
				AddRow(4, "Harry Potter and the Goblet of Fire", "J.K. Rowling").
				AddRow(5, "Harry Potter and the Order of the Phoenix", "J.K. Rowling").
				AddRow(6, "Harry Potter and the Half-Blood Prince", "J.K. Rowling").
				AddRow(7, "Harry Potter and the Deathly Hallows", "J.K. Rowling"))

			query := `
				{
					books {
						id
						title
						author {
							name
						}
					}
				}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"books": [
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 1,
				"title": "Harry Potter and the Philosopher's Stone"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 2,
				"title": "Harry Potter and the Chamber of Secrets"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 3,
				"title": "Harry Potter and the Prisoner of Azkaban"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 4,
				"title": "Harry Potter and the Goblet of Fire"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 5,
				"title": "Harry Potter and the Order of the Phoenix"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 6,
				"title": "Harry Potter and the Half-Blood Prince"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 7,
				"title": "Harry Potter and the Deathly Hallows"
			}
		]
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books with author name and reviews", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id)`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling").
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			mock.ExpectQuery(QuoteMeta(`SELECT r.book_id, r.star, r.body FROM reviews r WHERE r.book_id = ANY($1)`)).WithArgs("{1,2}").WillReturnRows(sqlmock.NewRows(
				[]string{"book_id", "star", "body"}).
				AddRow(1, 5, "The Boy Who Lived").
				AddRow(1, 4, "The stone that must be destroyed").
				AddRow(2, 5, "The Girl Who Kill"))

			query := `
				{
					books {
						id
						title
						author {
							name
						}
						reviews {
							star
							body
						}
					}
				}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"books": [
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 1,
				"reviews": [
					{
						"body": "The Boy Who Lived",
						"star": 5
					},
					{
						"body": "The stone that must be destroyed",
						"star": 4
					}
				],
				"title": "Harry Potter and the Philosopher's Stone"
			},
			{
				"author": {
					"name": "J.K. Rowling"
				},
				"id": 2,
				"reviews": [
					{
						"body": "The Girl Who Kill",
						"star": 5
					}
				],
				"title": "Harry Potter and the Chamber of Secrets"
			}
		]
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books by title", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title FROM books b WHERE b.title LIKE $1`)).WithArgs("Harry Potter %").WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			query := `
			{
				books(title_like: "Harry Potter %") {
					id
					title
				}
			}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"books": [
			{
				"id": 1,
				"title": "Harry Potter and the Philosopher's Stone"
			},
			{
				"id": 2,
				"title": "Harry Potter and the Chamber of Secrets"
			}
		]
	}
}`, string(rJSON))
		})

		t.Run("find books by author name", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE a.name LIKE $1`)).WithArgs("%Rowling").WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			query := `
			{
				books(author_like: "%Rowling") {
					id
					title
				}
			}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"books": [
			{
				"id": 1,
				"title": "Harry Potter and the Philosopher's Stone"
			},
			{
				"id": 2,
				"title": "Harry Potter and the Chamber of Secrets"
			}
		]
	}
}`, string(rJSON))
		})

		t.Run("find books by title and author name", func(t *testing.T) {
			mock.ExpectQuery(QuoteMeta(`SELECT b.id, b.title FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE b.title LIKE $1 AND a.name LIKE $2`)).WithArgs("Harry Potter %", "%Rowling").WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			query := `
			{
				books(title_like: "Harry Potter %", author_like: "%Rowling") {
					id
					title
				}  
			}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
			}

			rJSON, _ := json.MarshalIndent(r, "", "\t")
			assert.Equal(t, `{
	"data": {
		"books": [
			{
				"id": 1,
				"title": "Harry Potter and the Philosopher's Stone"
			},
			{
				"id": 2,
				"title": "Harry Potter and the Chamber of Secrets"
			}
		]
	}
}`, string(rJSON))
		})
	}
}
