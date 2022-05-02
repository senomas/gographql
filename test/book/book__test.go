package test_book

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/graphql-go/graphql"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/senomas/gographql/models"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
	"github.com/stretchr/testify/assert"
)

func TestBook_DB(t *testing.T) {
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
			Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: models.BookQueries(sqlDB, graphql.Fields{})}),
			Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: models.AuthorMutations(sqlDB, models.ReviewMutations(sqlDB, models.BookMutations(sqlDB, graphql.Fields{})))}),
		}
		var schema graphql.Schema
		if s, err := graphql.NewSchema(schemaConfig); err != nil {
			log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
		} else {
			schema = s
		}

		t.Run("create author J.K. Rowling", func(t *testing.T) {
			query := `
			mutation {
				createAuthor(name: "J.K. Rowling") {
					id
					name
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
		"createAuthor": {
			"id": 1,
			"name": "J.K. Rowling"
		}
	}
}`, string(rJSON))
		})

		t.Run("create book Harry Potter and the Philosopher's Stone", func(t *testing.T) {
			query := `
			mutation {
				createBook(title: "Harry Potter and the Philosopher's Stone", author: "J.K. Rowling") {
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
		"createBook": {
			"id": 1,
			"title": "Harry Potter and the Philosopher's Stone"
		}
	}
}`, string(rJSON))
		})

		t.Run("create book Harry Potter and the Philosopher's Stone Review 1", func(t *testing.T) {
			query := `
			mutation {
				createReview(book_id: 1, star: 5, body: "The Boy Who Lived") {
					id
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
		"createReview": {
			"id": 1
		}
	}
}`, string(rJSON))
		})

		t.Run("create book Harry Potter and the Philosopher's Stone Review 2", func(t *testing.T) {
			query := `
			mutation {
				createReview(book_id: 1, star: 4, body: "The stone that must be destroyed") {
					id
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
		"createReview": {
			"id": 2
		}
	}
}`, string(rJSON))
		})

		t.Run("create book Harry Potter and the Chamber of Secrets", func(t *testing.T) {
			query := `
			mutation {
				createBook(title: "Harry Potter and the Chamber of Secrets", author: "J.K. Rowling") {
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
		"createBook": {
			"id": 2,
			"title": "Harry Potter and the Chamber of Secrets"
		}
	}
}`, string(rJSON))
		})

		t.Run("create book Harry Potter and the Chamber of Secrets Review 1", func(t *testing.T) {
			query := `
			mutation {
				createReview(book_id: 2, star: 5, body: "The Girl Who Kill") {
					id
					book_id
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
		"createReview": {
			"book_id": 2,
			"id": 3
		}
	}
}`, string(rJSON))
		})

		t.Run("find book by id", func(t *testing.T) {
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
		})

		t.Run("find books with author name and reviews", func(t *testing.T) {
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
		})

	}
}

func QuoteMeta(r string) string {
	return "^" + regexp.QuoteMeta(r) + "$"
}
