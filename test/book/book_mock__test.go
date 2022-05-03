package test_book

import (
	"database/sql/driver"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/graphql-go/graphql"
	"github.com/senomas/gographql/models"
	"github.com/senomas/gographql/test"
	"github.com/stretchr/testify/assert"
)

func TestBook(t *testing.T) {
	if sqlDB, mock, err := sqlmock.New(); err != nil {
		t.Fatal("init SQLMock Error", err)
	} else {
		defer sqlDB.Close()

		schemaConfig := graphql.SchemaConfig{
			Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: models.CreateFields(models.BookQueries)}),
			Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: models.CreateFields(models.AuthorMutations, models.ReviewMutations, models.BookMutations)}),
		}
		var schema graphql.Schema
		if s, err := graphql.NewSchema(schemaConfig); err != nil {
			log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
		} else {
			schema = s
		}

		testQL, _ := test.QLTest(t, schema, models.NewContext(sqlDB))

		t.Run("create book with author_id", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(test.QuoteMeta(`INSERT INTO books (author_id, title) VALUES ($1, $2) RETURNING id`)).WithArgs(1, "Harry Potter and the Philosopher's Stone").WillReturnRows(sqlmock.NewRows(
				[]string{"id"}).
				AddRow(1))
			mock.ExpectCommit()
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title FROM books b WHERE b.id = $1 LIMIT 1`)).WithArgs(1).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone"))

			testQL(`mutation {
				createBook(title: "Harry Potter and the Philosopher's Stone", author_id: 1) {
					id
					title
				}
			}`, `{
				"data": {
					"createBook": {
						"id": 1,
						"title": "Harry Potter and the Philosopher's Stone"
					}
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books by id", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title FROM books b WHERE b.id = $1 LIMIT 1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			testQL(`{
				book(id: 2) {
					id
					title
				}
			}`, `{
				"data": {
					"book": {
						"id": 2,
						"title": "Harry Potter and the Chamber of Secrets"
					}
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find book with author name by id", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE b.id = $1 LIMIT 1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			testQL(`{
				book(id: 2) {
					id
					title
					author {
						name
					}
				}
			}`, `{
				"data": {
					"book": {
						"author": {
							"name": "J.K. Rowling"
						},
						"id": 2,
						"title": "Harry Potter and the Chamber of Secrets"
					}
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find book with author name and reviews by id", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE b.id = $1 LIMIT 1`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			mock.ExpectQuery(test.QuoteMeta(`SELECT r.book_id, r.star, r.body FROM reviews r WHERE r.book_id = $1 LIMIT 10`)).WithArgs(2).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "star", "body"}).
				AddRow(2, 5, "The Boy Who Lived").
				AddRow(2, 4, "The stone that must be destroyed"))

			testQL(`{
				book(id: 2) {
					id
					title
					author {
						name
					}
					reviews(query_limit: 10) {
						star
						body
					}
				}
			}`, `{
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
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books with author name", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id)`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling").
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling").
				AddRow(3, "Harry Potter and the Prisoner of Azkaban", "J.K. Rowling").
				AddRow(4, "Harry Potter and the Goblet of Fire", "J.K. Rowling").
				AddRow(5, "Harry Potter and the Order of the Phoenix", "J.K. Rowling").
				AddRow(6, "Harry Potter and the Half-Blood Prince", "J.K. Rowling").
				AddRow(7, "Harry Potter and the Deathly Hallows", "J.K. Rowling"))

			testQL(`{
				books {
					id
					title
					author {
						name
					}
				}
			}`, `{
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
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find limit books with author name", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) LIMIT 1`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling"))

			testQL(`{
				books(query_limit: 1) {
					id
					title
					author {
						name
					}
				}
			}`, `{
				"data": {
					"books": [
						{
							"author": {
								"name": "J.K. Rowling"
							},
							"id": 1,
							"title": "Harry Potter and the Philosopher's Stone"
						}
					]
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find offset books with author name", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) OFFSET 1`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling"))

			testQL(`{
				books(query_offset: 1) {
					id
					title
					author {
						name
					}
				}
			}`, `{
				"data": {
					"books": [
						{
							"author": {
								"name": "J.K. Rowling"
							},
							"id": 1,
							"title": "Harry Potter and the Philosopher's Stone"
						}
					]
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find limit offset books with author name", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id) LIMIT 1 OFFSET 1`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling"))

			testQL(`{
				books(query_limit: 1, query_offset: 1) {
					id
					title
					author {
						name
					}
				}
			}`, `{
				"data": {
					"books": [
						{
							"author": {
								"name": "J.K. Rowling"
							},
							"id": 1,
							"title": "Harry Potter and the Philosopher's Stone"
						}
					]
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books with author name and reviews", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id)`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling").
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			mock.ExpectQuery(test.QuoteMeta(`SELECT r.book_id, r.star, r.body FROM reviews r WHERE r.book_id = ANY($1)`)).WithArgs("{1,2}").WillReturnRows(sqlmock.NewRows(
				[]string{"book_id", "star", "body"}).
				AddRow(1, 5, "The Boy Who Lived").
				AddRow(1, 4, "The stone that must be destroyed").
				AddRow(2, 5, "The Girl Who Kill"))

			testQL(`{
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
			}`, `{
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
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books with author name and reviews limit", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title, a.name FROM (books b LEFT JOIN authors a ON b.author_id = a.id)`)).WithArgs([]driver.Value{}...).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title", "author_name"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone", "J.K. Rowling").
				AddRow(2, "Harry Potter and the Chamber of Secrets", "J.K. Rowling"))

			mock.ExpectQuery(test.QuoteMeta(`SELECT r.book_id, r.star, r.body FROM reviews r WHERE r.book_id = ANY($1) LIMIT 10`)).WithArgs("{1,2}").WillReturnRows(sqlmock.NewRows(
				[]string{"book_id", "star", "body"}).
				AddRow(1, 5, "The Boy Who Lived").
				AddRow(1, 4, "The stone that must be destroyed").
				AddRow(2, 5, "The Girl Who Kill"))

			testQL(`{
				books {
					id
					title
					author {
						name
					}
					reviews(query_limit: 10) {
						star
						body
					}
				}
			}`, `{
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
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("find books by title", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title FROM books b WHERE b.title LIKE $1`)).WithArgs("Harry Potter %").WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			testQL(`{
				books(title_like: "Harry Potter %") {
					id
					title
				}
			}`, `{
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
			}`)
		})

		t.Run("find books by author name", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE a.name LIKE $1`)).WithArgs("%Rowling").WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			testQL(`{
				books(author_like: "%Rowling") {
					id
					title
				}
			}`, `{
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
			}`)
		})

		t.Run("find books by title and author name", func(t *testing.T) {
			mock.ExpectQuery(test.QuoteMeta(`SELECT b.id, b.title FROM (books b LEFT JOIN authors a ON b.author_id = a.id) WHERE b.title LIKE $1 AND a.name LIKE $2`)).WithArgs("Harry Potter %", "%Rowling").WillReturnRows(sqlmock.NewRows(
				[]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Philosopher's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			testQL(`{
				books(title_like: "Harry Potter %", author_like: "%Rowling") {
					id
					title
				}  
			}`, `{
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
			}`)
		})
	}
}
