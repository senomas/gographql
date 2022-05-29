package graph_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"sync"
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

func addContext(ds *graph.DataSource) client.Option {
	return func(bd *client.Request) {
		ctx := context.WithValue(context.TODO(), graph.Context_DataSource, ds)
		bd.HTTP = bd.HTTP.WithContext(ctx)
	}
}

func TestTodo(t *testing.T) {
	cfg := generated.Config{Resolvers: &graph.Resolver{}}
	cfg.Directives.Gorm = graph.Directive_Gorm
	h := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))
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
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM "books"`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(4))
			mock.ExpectQuery(QuoteMeta(`
            SELECT "books"."id","books"."title" FROM "books" LIMIT 10
         `)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets").
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(4, "Harry Potter and the Snake Dictionary"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books {
            count
            list {
               id
               title
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 4,
				List: []*model.Book{
					{
						ID:    1,
						Title: "Harry Potter and the Sorcerer's Stone",
					},
					{
						ID:    2,
						Title: "Harry Potter and the Chamber of Secrets",
					},
					{
						ID:    3,
						Title: "Harry Potter and the Book of Evil",
					},
					{
						ID:    4,
						Title: "Harry Potter and the Snake Dictionary",
					},
				},
			},
		}, &resp)
	})

	t.Run("find books + authors", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM "books"`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(4))
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
					AddRow(1, "Harry Potter and the Sorcerer's Stone").
					AddRow(2, "Harry Potter and the Chamber of Secrets").
					AddRow(3, "Harry Potter and the Book of Evil").
					AddRow(4, "Harry Potter and the Snake Dictionary"))

			args := NewArrayIntArgs(1, 2, 3, 4)
			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id 
            WHERE book_authors.book_id IN ($1,$2,$3,$4)
         `)).WithArgs(args, args, args, args).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(1, 1, "J.K. Rowling").
				AddRow(2, 1, "J.K. Rowling").
				AddRow(3, 2, "Lord Voldermort").
				AddRow(4, 2, "Lord Voldermort").
				AddRow(4, 3, "Salazar Slitherin"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books {
            count
            list {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 4,
				List: []*model.Book{
					{
						ID:    1,
						Title: "Harry Potter and the Sorcerer's Stone",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
					{
						ID:    2,
						Title: "Harry Potter and the Chamber of Secrets",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
					{
						ID:    3,
						Title: "Harry Potter and the Book of Evil",
						Authors: []*model.Author{
							{
								ID:   2,
								Name: "Lord Voldermort",
							},
						},
					},
					{
						ID:    4,
						Title: "Harry Potter and the Snake Dictionary",
						Authors: []*model.Author{
							{
								ID:   2,
								Name: "Lord Voldermort",
							},
							{
								ID:   3,
								Name: "Salazar Slitherin",
							},
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("find book limit 1", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`
            SELECT count(*) FROM "books"
         `)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(4))
			mock.ExpectQuery(QuoteMeta(`
            SELECT "books"."id","books"."title" FROM "books" LIMIT 1
         `)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone"))

			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id 
            WHERE book_authors.book_id IN ($1)
         `)).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(1, 1, "J.K. Rowling"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books(limit: 1) {
            count
            list {
               id
               title
               authors {
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 4,
				List: []*model.Book{
					{
						ID:    1,
						Title: "Harry Potter and the Sorcerer's Stone",
						Authors: []*model.Author{
							{
								Name: "J.K. Rowling",
							},
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("find books filter by title and author_name", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`
            SELECT count(*) FROM "books" 
            WHERE books.title LIKE $1 AND books.id IN (SELECT book_id FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE authors.name = $2)
         `)).WithArgs("%Harry Potter%", "Lord Voldermort").WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(2))

			mock.ExpectQuery(QuoteMeta(`
            SELECT "books"."id","books"."title" FROM "books" 
            WHERE books.title LIKE $1 AND books.id IN (
                  SELECT book_id FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE authors.name = $2
            ) LIMIT 10
         `)).WithArgs("%Harry Potter%", "Lord Voldermort").WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(4, "Harry Potter and the Snake Dictionary"))

			args := NewArrayIntArgs(3, 4)
			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id 
            WHERE book_authors.book_id IN ($1,$2)
         `)).WithArgs(args, args).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(3, 2, "Lord Voldermort").
				AddRow(4, 2, "Lord Voldermort").
				AddRow(4, 3, "Salazar Slitherin"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books(filter: {
               title: {
                  op: LIKE
                  value: "%Harry Potter%"
               }
               author_name: {
                  op: EQ
                  value: "Lord Voldermort"
               }
            }
         ) {
            count
            list {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 2,
				List: []*model.Book{
					{
						ID:    3,
						Title: "Harry Potter and the Book of Evil",
						Authors: []*model.Author{
							{
								ID:   2,
								Name: "Lord Voldermort",
							},
						},
					},
					{
						ID:    4,
						Title: "Harry Potter and the Snake Dictionary",
						Authors: []*model.Author{
							{
								ID:   2,
								Name: "Lord Voldermort",
							},
							{
								ID:   3,
								Name: "Salazar Slitherin",
							},
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("find books filter by title not and author_name not", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`
            SELECT count(*) FROM "books" WHERE books.title NOT LIKE $1 AND books.id NOT IN (
               SELECT book_id FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE authors.name = $2
            )`)).WithArgs("%Stone%", "Lord Voldermort").WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(1))

			mock.ExpectQuery(QuoteMeta(`
            SELECT "books"."id","books"."title" FROM "books" WHERE books.title NOT LIKE $1 AND books.id NOT IN (
               SELECT book_id FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE authors.name = $2
            ) LIMIT 10`)).WithArgs("%Stone%", "Lord Voldermort").WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(2, "Harry Potter and the Chamber of Secrets"))

			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors"
            JOIN book_authors ON authors.id = book_authors.author_id WHERE book_authors.book_id IN ($1)
            `)).WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(2, 1, "J.K. Rowling"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books(filter: {
               title: {
                  op: NOT_LIKE
                  value: "%Stone%"
               }
               author_name: {
                  op: NOT_EQ
                  value: "Lord Voldermort"
               }
            }
         ) {
            count
            list {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 1,
				List: []*model.Book{
					{
						ID:    2,
						Title: "Harry Potter and the Chamber of Secrets",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("find books filter unknown author_name", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`
            SELECT count(*) FROM "books" WHERE books.id IN (SELECT book_id FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE authors.name = $1)
            `)).WithArgs("Harry Potter").WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(0))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books(filter: {
               author_name: {
                  op: EQ
                  value: "Harry Potter"
               }
            }
         ) {
            count
            list {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 0,
				List:  []*model.Book{},
			},
		}, &resp)
	})

	t.Run("find books + reviews", func(t *testing.T) {
		if mock != nil {
			reviewArgs := NewArrayIntArgs(1, 2, 3, 4)
			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM "books"`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(4))
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets").
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(4, "Harry Potter and the Snake Dictionary"))
			mock.ExpectQuery(QuoteMeta(`SELECT "book_id","id","star","text" FROM "reviews" WHERE book_id IN ($1,$2,$3,$4)`)).
				WithArgs(reviewArgs, reviewArgs, reviewArgs, reviewArgs).
				WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "star", "text"}).
					AddRow(1, 1, 5, "The Boy Who Live").
					AddRow(2, 2, 5, "The Girl Who Kill").
					AddRow(3, 3, 1, "Fake Books").
					AddRow(4, 1, 3, "The Man With Funny Hat"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books {
            count
               list {
                  id
               title
               reviews {
                  id
                  star
                  text
               }
            }
            }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 4,
				List: []*model.Book{
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
								Star: 3,
								Text: "The Man With Funny Hat",
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
					{
						ID:      4,
						Title:   "Harry Potter and the Snake Dictionary",
						Reviews: []*model.Review{},
					},
				},
			},
		}, &resp)
	})

	t.Run("find books + reviews filter by star", func(t *testing.T) {
		if mock != nil {
			reviewArgs := NewArrayIntArgs(1, 2, 3, 4)
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets").
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(4, "Harry Potter and the Snake Dictionary"))
			mock.ExpectQuery(QuoteMeta(`SELECT "book_id","id","star","text" FROM "reviews" WHERE book_id IN ($1,$2,$3,$4) AND "reviews"."star" >= $5`)).
				WithArgs(reviewArgs, reviewArgs, reviewArgs, reviewArgs, 3).WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "star", "text"}).
				AddRow(1, 1, 5, "The Boy Who Live").
				AddRow(2, 2, 5, "The Girl Who Kill").
				AddRow(4, 1, 3, "The Man With Funny Hat"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books {
            list {
               id
               title
               reviews(filter: { star: { min: 3}}) {
                  id
                  star
                  text
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				List: []*model.Book{
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
								Star: 3,
								Text: "The Man With Funny Hat",
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
					{
						ID:      4,
						Title:   "Harry Potter and the Snake Dictionary",
						Reviews: []*model.Review{},
					},
				},
			},
		}, &resp)
	})

	t.Run("find books filter by title and review.star", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`
            SELECT DISTINCT "books"."id","books"."title" FROM "books" JOIN reviews ON books.id = reviews.book_id WHERE books.title LIKE $1 AND "reviews"."star" >= $2 LIMIT 10
            `)).WithArgs("%Harry Potter%", 3).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books(filter: {title: {op: LIKE, value: "%Harry Potter%"}, star: {min: 3}}) {
            list {
               id
               title
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				List: []*model.Book{
					{
						ID:    1,
						Title: "Harry Potter and the Sorcerer's Stone",
					},
					{
						ID:    2,
						Title: "Harry Potter and the Chamber of Secrets",
					},
				},
			},
		}, &resp)
	})

	t.Run("find books concurrent", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets").
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(4, "Harry Potter and the Snake Dictionary"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}

		ctx := addContext(graph.NewDataSource(db))
		var wg sync.WaitGroup
		test := func() {
			defer wg.Done()

			var resp respType
			c.MustPost(`{
         books {
            list {
               id
               title
            }
         }
      }`, &resp, ctx)

			JsonMatch(t, &respType{
				Books: model.BookList{
					List: []*model.Book{
						{
							ID:    1,
							Title: "Harry Potter and the Sorcerer's Stone",
						},
						{
							ID:    2,
							Title: "Harry Potter and the Chamber of Secrets",
						},
						{
							ID:    3,
							Title: "Harry Potter and the Book of Evil",
						},
						{
							ID:    4,
							Title: "Harry Potter and the Snake Dictionary",
						},
					},
				},
			}, &resp)
		}

		wg.Add(2)
		go test()
		go test()

		wg.Wait()
	})

	t.Run("create review", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE id = $1 LIMIT 1`)).WithArgs(3).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).
				AddRow(3, "Harry Potter and the Book of Evil", 2))
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO "reviews" ("star","text","book_id") VALUES ($1,$2,$3) RETURNING "id"`)).WithArgs(5, "Tom Riddle", 3).WillReturnRows(sqlmock.NewRows([]string{"id"}).
				AddRow(5))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" WHERE books.id IN ($1)`)).WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(3, "Harry Potter and the Book of Evil"))

			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE book_authors.book_id IN ($1)
            `)).WithArgs(3).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(3, 2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type CreateReview struct {
			ID   int
			Star int
			Text string
			Book model.Book
		}
		type respType struct {
			CreateReview CreateReview
		}
		var resp respType
		c.MustPost(`mutation {
         createReview(input: {
            book_id: 3
            star: 5
            text: "Tom Riddle"
         }) {
            id
            star
            text
            book {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			CreateReview: CreateReview{
				ID:   5,
				Star: 5,
				Text: "Tom Riddle",
				Book: model.Book{
					ID:    3,
					Title: "Harry Potter and the Book of Evil",
					Authors: []*model.Author{
						{
							ID:   2,
							Name: "Lord Voldermort",
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("create book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE name IN ($1,$2)`)).WithArgs("J.K. Rowling", "Albus Dumbledore").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(1, "J.K. Rowling").
				AddRow(4, "Albus Dumbledore"))
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO "books" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("Harry Potter and the Unknown").WillReturnRows(sqlmock.NewRows([]string{"id"}).
				AddRow(5))
			mock.ExpectExec(QuoteMeta(`INSERT INTO "book_authors" ("book_id","author_id") VALUES ($1,$2),($3,$4) ON CONFLICT DO NOTHING`)).WithArgs(5, 1, 5, 4).WillReturnResult(driver.RowsAffected(2))
			mock.ExpectCommit()

			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE book_authors.book_id IN ($1)
            `)).WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(5, 1, "J.K. Rowling").
				AddRow(5, 4, "Albus Dumbledore"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type CreateBook struct {
			ID      int
			Title   string
			Authors []*model.Author
		}
		type respType struct {
			CreateBook CreateBook
		}
		var resp respType
		c.MustPost(`mutation {
         createBook(input: {
            title: "Harry Potter and the Unknown"
            authors_name: ["J.K. Rowling", "Albus Dumbledore"]
         }) {
            id
            title
            authors {
               id
               name
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			CreateBook: CreateBook{
				ID:    5,
				Title: "Harry Potter and the Unknown",
				Authors: []*model.Author{
					{
						ID:   1,
						Name: "J.K. Rowling",
					},
					{
						ID:   4,
						Name: "Albus Dumbledore",
					},
				},
			},
		}, &resp)
	})

	t.Run("create duplicate book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE name IN ($1,$2)`)).WithArgs("J.K. Rowling", "Albus Dumbledore").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(1, "J.K. Rowling").
				AddRow(4, "Albus Dumbledore"))
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO "books" ("title") VALUES ($1) RETURNING "id"`)).
				WithArgs("Harry Potter and the Unknown").
				WillReturnError(errors.New(`ERROR: duplicate key value violates unique constraint "books_title_key" (SQLSTATE 23505)`))
			mock.ExpectRollback()
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type CreateBook struct {
			ID     int
			Title  string
			Author model.Author
		}
		type respType struct {
			CreateBook CreateBook
		}
		var resp respType
		err := c.Post(`mutation {
         createBook(input: {
            title: "Harry Potter and the Unknown"
            authors_name: ["J.K. Rowling", "Albus Dumbledore"]
         }) {
            id
            title
            authors {
               id
               name
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `duplicate key books.title \"Harry Potter and the Unknown\"`)
	})

	t.Run("update book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE books.id = $1 LIMIT 1`)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(4, "Harry Potter and the Unknown"))
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "authors" WHERE name IN ($1,$2)`)).WithArgs("Albus Dumbledore", "Salazar Slitherin").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(4, "Salazar Slitherin").
				AddRow(5, "Albus Dumbledore"))
			mock.ExpectExec(QuoteMeta(`DELETE FROM book_authors WHERE book_id = $1 AND author_id NOT IN ($2,$3)`)).WithArgs(4, 4, 5).WillReturnResult(driver.RowsAffected(1))
			mock.ExpectExec(QuoteMeta(`UPDATE "books" SET "title"=$1 WHERE "id" = $2`)).WithArgs("Harry Potter and the Fake Book", 4).WillReturnResult(driver.RowsAffected(1))
			mock.ExpectExec(QuoteMeta(`INSERT INTO "book_authors" ("book_id","author_id") VALUES ($1,$2),($3,$4) ON CONFLICT DO NOTHING`)).WithArgs(4, 4, 4, 5).WillReturnResult(driver.RowsAffected(1))
			mock.ExpectCommit()

			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id WHERE book_authors.book_id IN ($1)
            `)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(4, 3, "Salazar Slitherin").
				AddRow(4, 4, "Albus Dumbledore"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type UpdateBook struct {
			ID      int
			Title   string
			Authors []*model.Author
		}
		type respType struct {
			UpdateBook UpdateBook
		}
		var resp respType
		c.MustPost(`mutation {
         updateBook(input: {
            id: 4
            title: "Harry Potter and the Fake Book"
            authors_name: ["Albus Dumbledore", "Salazar Slitherin"]
         }) {
            id
            title
            authors {
               id
               name
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			UpdateBook: UpdateBook{
				ID:    4,
				Title: "Harry Potter and the Fake Book",
				Authors: []*model.Author{
					{
						ID:   3,
						Name: "Salazar Slitherin",
					},
					{
						ID:   4,
						Name: "Albus Dumbledore",
					},
				},
			},
		}, &resp)
	})

	t.Run("find updated books", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM "books"`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets").
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(5, "Harry Potter and the Unknown").
				AddRow(4, "Harry Potter and the Fake Book"))
			args := NewArrayIntArgs(1, 2, 3, 4, 5)
			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id 
            WHERE book_authors.book_id IN ($1,$2,$3,$4,$5)`)).WithArgs(args, args, args, args, args).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(1, 1, "J.K. Rowling").
				AddRow(2, 1, "J.K. Rowling").
				AddRow(3, 2, "Lord Voldermort").
				AddRow(4, 3, "Salazar Slitherin").
				AddRow(4, 4, "Albus Dumbledore").
				AddRow(5, 1, "J.K. Rowling").
				AddRow(5, 4, "Albus Dumbledore"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books {
            count
            list {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 5,
				List: []*model.Book{
					{
						ID:    1,
						Title: "Harry Potter and the Sorcerer's Stone",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
					{
						ID:    2,
						Title: "Harry Potter and the Chamber of Secrets",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
					{
						ID:    3,
						Title: "Harry Potter and the Book of Evil",
						Authors: []*model.Author{
							{
								ID:   2,
								Name: "Lord Voldermort",
							},
						},
					},
					{
						ID:    5,
						Title: "Harry Potter and the Unknown",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
							{
								ID:   4,
								Name: "Albus Dumbledore",
							},
						},
					},
					{
						ID:    4,
						Title: "Harry Potter and the Fake Book",
						Authors: []*model.Author{
							{
								ID:   3,
								Name: "Salazar Slitherin",
							},
							{
								ID:   4,
								Name: "Albus Dumbledore",
							},
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("update book duplicate", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE books.id = $1 LIMIT 1`)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(4, "Harry Potter and the Unknown"))
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`UPDATE "books" SET "title"=$1 WHERE "id" = $2`)).WithArgs("Harry Potter and the Sorcerer's Stone", 4).
				WillReturnError(errors.New(`ERROR: duplicate key value violates unique constraint "books_title_key" (SQLSTATE 23505)`))
			mock.ExpectRollback()
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type UpdateBook struct {
			ID     int
			Title  string
			Author model.Author
		}
		type respType struct {
			UpdateBook UpdateBook
		}
		var resp respType
		err := c.Post(`mutation {
         updateBook(input: {
            id: 4
            title: "Harry Potter and the Sorcerer's Stone"
         }) {
            id
            title
            authors {
               id
               name
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `duplicate key books.title \"Harry Potter and the Sorcerer's Stone\"`)
	})

	t.Run("update unknown book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE books.id = $1 LIMIT 1`)).WithArgs(999).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id", "Author__id", "Author__name"}))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type UpdateBook struct {
			ID     int
			Title  string
			Author model.Author
		}
		type respType struct {
			UpdateBook UpdateBook
		}
		var resp respType
		err := c.Post(`mutation {
         updateBook(input: {
            id: 999
            title: "Harry Potter and the Sorcerer's Stone"
         }) {
            id
            title
            authors {
               id
               name
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `book with id '999' does not exist`)
	})

	t.Run("delete book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE books.id = $1 LIMIT 1`)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(4, "Harry Potter and the Fake Book"))
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`DELETE FROM "books" WHERE "books"."id" = $1`)).WithArgs(4).WillReturnResult(driver.RowsAffected(1))
			mock.ExpectCommit()
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type DeleteBook struct {
			ID      int
			Title   string
			Authors []*model.Author
		}
		type respType struct {
			DeleteBook DeleteBook
		}
		var resp respType
		c.MustPost(`mutation {
         deleteBook(id: 4) {
            id
            title
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			DeleteBook: DeleteBook{
				ID:    4,
				Title: "Harry Potter and the Fake Book",
			},
		}, &resp)
	})

	t.Run("find deleted books", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM "books"`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(4))
			mock.ExpectQuery(QuoteMeta(`SELECT "books"."id","books"."title" FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).
				AddRow(1, "Harry Potter and the Sorcerer's Stone").
				AddRow(2, "Harry Potter and the Chamber of Secrets").
				AddRow(3, "Harry Potter and the Book of Evil").
				AddRow(5, "Harry Potter and the Unknown"))

			args := NewArrayIntArgs(1, 2, 3, 5)
			mock.ExpectQuery(QuoteMeta(`
            SELECT "book_authors"."book_id","authors"."id","authors"."name" FROM "authors" JOIN book_authors ON authors.id = book_authors.author_id 
            WHERE book_authors.book_id IN ($1,$2,$3,$4)`)).WithArgs(args, args, args, args).WillReturnRows(sqlmock.NewRows([]string{"book_id", "id", "name"}).
				AddRow(1, 1, "J.K. Rowling").
				AddRow(2, 1, "J.K. Rowling").
				AddRow(3, 2, "Lord Voldermort").
				AddRow(5, 1, "J.K. Rowling").
				AddRow(5, 4, "Albus Dumbledore"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books model.BookList
		}
		var resp respType
		c.MustPost(`{
         books {
            count
            list {
               id
               title
               authors {
                  id
                  name
               }
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: model.BookList{
				Count: 4,
				List: []*model.Book{
					{
						ID:    1,
						Title: "Harry Potter and the Sorcerer's Stone",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
					{
						ID:    2,
						Title: "Harry Potter and the Chamber of Secrets",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
						},
					},
					{
						ID:    3,
						Title: "Harry Potter and the Book of Evil",
						Authors: []*model.Author{
							{
								ID:   2,
								Name: "Lord Voldermort",
							},
						},
					},
					{
						ID:    5,
						Title: "Harry Potter and the Unknown",
						Authors: []*model.Author{
							{
								ID:   1,
								Name: "J.K. Rowling",
							},
							{
								ID:   4,
								Name: "Albus Dumbledore",
							},
						},
					},
				},
			},
		}, &resp)
	})

	t.Run("delete unknown book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE books.id = $1 LIMIT 1`)).WithArgs(999).
				WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id", "Author__id", "Author__name"}))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type DeleteBook struct {
			ID      int
			Title   string
			Authors []*model.Author
		}
		type respType struct {
			DeleteBook DeleteBook
		}
		var resp respType
		err := c.Post(`mutation {
         deleteBook(id: 999) {
            id
            title
            authors {
               id
               name
            }
         }
      }`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `book with id '999' does not exist`)
	})
}
