package test_book

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/graphql-go/graphql"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/senomas/gographql/models"
	"github.com/senomas/gographql/test"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
)

func TestBook_DB(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dsn := "postgresql://demo:password@localhost/postgres?sslmode=disable&TimeZone=Asia/Jakarta"

	if sqlDBDirect, err := sql.Open("postgres", dsn); err != nil {
		t.Fatal("init SQLMock Error", err)
	} else {
		defer sqlDBDirect.Close()

		zerolog.SetGlobalLevel(zerolog.WarnLevel)
		loggerAdapter := zerologadapter.New(zerolog.New(os.Stdout))
		sqlDB := sqldblogger.OpenDriver(dsn, sqlDBDirect.Driver(), loggerAdapter)
		if err := sqlDB.Ping(); err != nil {
			log.Fatalf("Failed to ping database, err %v", err)
		}

		if err := models.SetupDevDatabase(sqlDB); err != nil {
			log.Fatalf("Failed to setupDatabase, err %v", err)
		}

		schemaConfig := graphql.SchemaConfig{
			Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: models.BookQueries(graphql.Fields{})}),
			Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: models.AuthorMutations(models.ReviewMutations(models.BookMutations(graphql.Fields{})))}),
		}
		var schema graphql.Schema
		if s, err := graphql.NewSchema(schemaConfig); err != nil {
			log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
		} else {
			schema = s
		}

		eval, _ := test.GqlTest(t, schema, models.NewContext(sqlDB))

		t.Run("create author J.K. Rowling", func(t *testing.T) {
			eval(`mutation {
				createAuthor(name: "J.K. Rowling") {
					id
					name
				}
			}`, `{
				"data": {
					"createAuthor": {
						"id": 1,
						"name": "J.K. Rowling"
					}
				}
			}`)
		})

		t.Run("create book Harry Potter and the Philosopher's Stone", func(t *testing.T) {
			eval(`mutation {
				createBook(title: "Harry Potter and the Philosopher's Stone", author: "J.K. Rowling") {
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
		})

		t.Run("create book Harry Potter and the Philosopher's Stone Review 1", func(t *testing.T) {
			eval(`mutation {
				createReview(book_id: 1, star: 5, body: "The Boy Who Lived") {
					id
				}
			}`, `{
				"data": {
					"createReview": {
						"id": 1
					}
				}
			}`)
		})

		t.Run("create book Harry Potter and the Philosopher's Stone Review 2", func(t *testing.T) {
			eval(`mutation {
				createReview(book_id: 1, star: 4, body: "The stone that must be destroyed") {
					id
				}
			}`, `{
				"data": {
					"createReview": {
						"id": 2
					}
				}
			}`)
		})

		t.Run("create book Harry Potter and the Chamber of Secrets", func(t *testing.T) {
			eval(`mutation {
				createBook(title: "Harry Potter and the Chamber of Secrets", author: "J.K. Rowling") {
					id
					title
				}
			}`, `{
				"data": {
					"createBook": {
						"id": 2,
						"title": "Harry Potter and the Chamber of Secrets"
					}
				}
			}`)
		})

		t.Run("create book Harry Potter and the Chamber of Secrets Review 1", func(t *testing.T) {
			eval(`mutation {
				createReview(book_id: 2, star: 5, body: "The Girl Who Kill") {
					id
					book_id
				}
			}`, `{
				"data": {
					"createReview": {
						"book_id": 2,
						"id": 3
					}
				}
			}`)
		})

		t.Run("create book Harry Potter and the Prisoner of Azkaban", func(t *testing.T) {
			eval(`mutation {
				createBook(title: "Harry Potter and the Prisoner of Azkaban", author: "J.K. Rowling") {
					id
					title
				}
			}`, `{
				"data": {
					"createBook": {
						"id": 3,
						"title": "Harry Potter and the Prisoner of Azkaban"
					}
				}
			}`)
		})

		t.Run("find book by id", func(t *testing.T) {
			eval(`{
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
		})

		t.Run("find books with author name and reviews", func(t *testing.T) {
			eval(`{
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
						},
						{
							"author": {
								"name": "J.K. Rowling"
							},
							"id": 3,
							"reviews": [],
							"title": "Harry Potter and the Prisoner of Azkaban"
						}
					]
				}
			}`)
		})

		t.Run("find limit books with author name", func(t *testing.T) {
			eval(`{
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
		})

		t.Run("find offset books with author name", func(t *testing.T) {
			eval(`{
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
							"id": 2,
							"title": "Harry Potter and the Chamber of Secrets"
						},
						{
							"author": {
								"name": "J.K. Rowling"
							},
							"id": 3,
							"title": "Harry Potter and the Prisoner of Azkaban"
						}
					]
				}
			}`)
		})

		t.Run("find limit offset books with author name", func(t *testing.T) {
			eval(`{
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
							"id": 2,
							"title": "Harry Potter and the Chamber of Secrets"
						}
					]
				}
			}`)
		})
	}
}
