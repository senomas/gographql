package graph_test

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/senomas/gographql/graph"
	"github.com/senomas/gographql/graph/generated"
	"github.com/senomas/gographql/graph/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func addContext(db *gorm.DB) client.Option {
	return func(bd *client.Request) {
		ctx := context.WithValue(context.TODO(), graph.Context_DataSource, graph.NewDataSource(db))
		bd.HTTP = bd.HTTP.WithContext(ctx)
	}
}

func TestTodo(t *testing.T) {

	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
	c := client.New(h)

	var db *gorm.DB
	var mock sqlmock.Sqlmock

	if _, _db, _mock, err := Setup(); err != nil {
		t.Fatalf("setup database error %v", err)
	} else {
		db = _db
		mock = _mock
	}

	t.Run("find books", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id","title","author_id" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2))
		mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE id = $1`)).WithArgs("{2,1}").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))

		type respType struct {
			Books []model.Book
		}
		var resp respType
		mock.MatchExpectationsInOrder(false)
		c.MustPost(`{
			books {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(db))
		mock.MatchExpectationsInOrder(true)

		JsonMatch(t, &respType{
			Books: []model.Book{
				{
					ID:    1,
					Title: "Harry Potter and the Sorcerer's Stone",
					Author: &model.Author{
						ID:   1,
						Name: "J.K. Rowling",
					},
				},
				{
					ID:    2,
					Title: "Harry Potter and the Chamber of Secrets",
					Author: &model.Author{
						ID:   1,
						Name: "J.K. Rowling",
					},
				},
				{
					ID:    3,
					Title: "Harry Potter and the Book of Evil",
					Author: &model.Author{
						ID:   2,
						Name: "Lord Voldermort",
					},
				},
			},
		}, &resp)
	})

	t.Run("find books with limit", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id","title","author_id" FROM "books" LIMIT 100`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2))
		mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE id = $1`)).WithArgs("{2,1}").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))

		type respType struct {
			Books []model.Book
		}
		var resp respType
		mock.MatchExpectationsInOrder(false)
		c.MustPost(`{
			books(query_limit: 100) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(db))
		mock.MatchExpectationsInOrder(true)

		JsonMatch(t, &respType{
			Books: []model.Book{
				{
					ID:    1,
					Title: "Harry Potter and the Sorcerer's Stone",
					Author: &model.Author{
						ID:   1,
						Name: "J.K. Rowling",
					},
				},
				{
					ID:    2,
					Title: "Harry Potter and the Chamber of Secrets",
					Author: &model.Author{
						ID:   1,
						Name: "J.K. Rowling",
					},
				},
				{
					ID:    3,
					Title: "Harry Potter and the Book of Evil",
					Author: &model.Author{
						ID:   2,
						Name: "Lord Voldermort",
					},
				},
			},
		}, &resp)
	})

	t.Run("find books filter by title and authorName", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id","title","author_id" FROM "books" WHERE title LIKE $1 AND author_id = SELECT id FROM "author" WHERE name LIKE $2 LIMIT 10`)).WithArgs("Harry Potter", "J.K. Rowling").WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2))
		mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE id = $1`)).WithArgs("{2,1}").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))

		type respType struct {
			Books []model.Book
		}
		var resp respType
		mock.MatchExpectationsInOrder(false)
		c.MustPost(`{
			books(title: "Harry Potter", authorName: "J.K. Rowling") {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(db))
		mock.MatchExpectationsInOrder(true)

		JsonMatch(t, &respType{
			Books: []model.Book{
				{
					ID:    1,
					Title: "Harry Potter and the Sorcerer's Stone",
					Author: &model.Author{
						ID:   1,
						Name: "J.K. Rowling",
					},
				},
				{
					ID:    2,
					Title: "Harry Potter and the Chamber of Secrets",
					Author: &model.Author{
						ID:   1,
						Name: "J.K. Rowling",
					},
				},
				{
					ID:    3,
					Title: "Harry Potter and the Book of Evil",
					Author: &model.Author{
						ID:   2,
						Name: "Lord Voldermort",
					},
				},
			},
		}, &resp)
	})

	t.Run("create book", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id" FROM "authors" WHERE name = $1 LIMIT 1`)).WithArgs("J.K. Rowling").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectBegin()
		mock.ExpectQuery(QuoteMeta(`INSERT INTO "books" ("title","author_id") VALUES ($1,$2) RETURNING "id"`)).WithArgs("Harry Potter and the Sorcerer's Stone", 1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
		mock.ExpectCommit()
		mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE id = $1`)).WithArgs("{1}").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling"))

		type CreateBook struct {
			ID     int
			Title  string
			Author model.Author
		}
		type respType struct {
			CreateBook CreateBook
		}
		var resp respType
		c.MustPost(`mutation {
			createBook(input: {
				title: "Harry Potter and the Sorcerer's Stone"
				authorName: "J.K. Rowling"
			}) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(db))
		assert.NoError(t, mock.ExpectationsWereMet())
		JsonMatch(t, &respType{
			CreateBook: CreateBook{
				ID:    5,
				Title: "Harry Potter and the Sorcerer's Stone",
				Author: model.Author{
					ID:   1,
					Name: "J.K. Rowling",
				},
			},
		}, &resp)
	})
}
