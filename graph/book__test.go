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
		mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id = $1`)).WithArgs("{2,1}").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))

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
		mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id = $1`)).WithArgs(&MatchPQArray{Value: "{1,2}"}).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))

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
		mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id = $1`)).WithArgs(&MatchPQArray{Value: "{1,2}"}).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))

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

	t.Run("find books with reviews", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id","title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(1, "Harry Potter and the Sorcerer's Stone").AddRow(2, "Harry Potter and the Chamber of Secrets").AddRow(3, "Harry Potter and the Book of Evil"))
		mock.ExpectQuery(QuoteMeta(`SELECT "id","star","text" FROM "reviews" WHERE book_id = $1`)).WithArgs(&MatchPQArray{Value: "{1,2,3}"}).WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "star", "text"}).AddRow(1, 1, 5, "The Boy Who Live").AddRow(2, 2, 5, "The Girl Who Kill").AddRow(3, 3, 1, "Fake Books").AddRow(4, 1, 5, "The Man Who Wear Turban"))

		type respType struct {
			Books []model.Book
		}
		var resp respType
		mock.MatchExpectationsInOrder(false)
		c.MustPost(`{
			books {
				id
				title
				reviews {
					id
					star
					text
				}
			}
		}`, &resp, addContext(db))
		mock.MatchExpectationsInOrder(true)

		JsonMatch(t, &respType{
			Books: []model.Book{
				{
					ID:    1,
					Title: "Harry Potter and the Sorcerer's Stone",
					Reviews: []*model.Review{
						{
							ID:   1,
							Star: 5,
							Text: "The Boy Who Live",
						},
						{
							ID:   4,
							Star: 5,
							Text: "The Man Who Wear Turban",
						},
					},
				},
				{
					ID:    2,
					Title: "Harry Potter and the Chamber of Secrets",
					Reviews: []*model.Review{
						{
							ID:   2,
							Star: 5,
							Text: "The Girl Who Kill",
						},
					},
				},
				{
					ID:    3,
					Title: "Harry Potter and the Book of Evil",
					Reviews: []*model.Review{
						{
							ID:   3,
							Star: 1,
							Text: "Fake Books",
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("find books with reviews (3 star min)", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id","title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(1, "Harry Potter and the Sorcerer's Stone").AddRow(2, "Harry Potter and the Chamber of Secrets").AddRow(3, "Harry Potter and the Book of Evil"))
		mock.ExpectQuery(QuoteMeta(`SELECT "id","star","text" FROM "reviews" WHERE book_id = $1 AND star >= $2`)).WithArgs(&MatchPQArray{Value: "{1,2,3}"}, 3).WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "star", "text"}).AddRow(1, 1, 5, "The Boy Who Live").AddRow(2, 2, 5, "The Girl Who Kill").AddRow(4, 1, 5, "The Man Who Wear Turban"))

		type respType struct {
			Books []model.Book
		}
		var resp respType
		mock.MatchExpectationsInOrder(false)
		c.MustPost(`{
			books {
				id
				title
				reviews(minStar: 3) {
					id
					star
					text
				}
			}
		}`, &resp, addContext(db))
		mock.MatchExpectationsInOrder(true)

		JsonMatch(t, &respType{
			Books: []model.Book{
				{
					ID:    1,
					Title: "Harry Potter and the Sorcerer's Stone",
					Reviews: []*model.Review{
						{
							ID:   1,
							Star: 5,
							Text: "The Boy Who Live",
						},
						{
							ID:   4,
							Star: 5,
							Text: "The Man Who Wear Turban",
						},
					},
				},
				{
					ID:    2,
					Title: "Harry Potter and the Chamber of Secrets",
					Reviews: []*model.Review{
						{
							ID:   2,
							Star: 5,
							Text: "The Girl Who Kill",
						},
					},
				},
				{
					ID:      3,
					Title:   "Harry Potter and the Book of Evil",
					Reviews: []*model.Review{},
				},
			},
		}, &resp)
	})

	t.Run("create book", func(t *testing.T) {
		mock.ExpectQuery(QuoteMeta(`SELECT "id" FROM "authors" WHERE name = $1 LIMIT 1`)).WithArgs("J.K. Rowling").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectBegin()
		mock.ExpectQuery(QuoteMeta(`INSERT INTO "books" ("title","author_id") VALUES ($1,$2) RETURNING "id"`)).WithArgs("Harry Potter and the Sorcerer's Stone", 1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
		mock.ExpectCommit()
		mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id = $1`)).WithArgs("{1}").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling"))

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
