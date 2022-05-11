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
		if mock != nil {
			authorArgs := NewArrayIntArgs(1, 2)
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title,books.author_id FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2))
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1,$2)`)).WithArgs(authorArgs, authorArgs).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

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
		if mock != nil {
			authorArgs := NewArrayIntArgs(1, 2)
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title,books.author_id FROM "books" LIMIT 100`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2))
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1,$2)`)).WithArgs(authorArgs, authorArgs).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books(limit: 100) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

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
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title,books.author_id FROM "books" WHERE books.title LIKE $1 AND author_id = (SELECT id FROM "authors" WHERE name = $2) LIMIT 10`)).WithArgs("%Harry Potter%", "J.K. Rowling").WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1))
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1)`)).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books(filter: {
					title: {
						op: LIKE
						value: "%Harry Potter%"
					}
					authorName: {
						op: EQ
						value: "J.K. Rowling"
					}
				}) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

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
			},
		}, &resp)
	})

	t.Run("find books + reviews", func(t *testing.T) {
		if mock != nil {
			reviewArgs := NewArrayIntArgs(1, 2, 3)
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(1, "Harry Potter and the Sorcerer's Stone").AddRow(2, "Harry Potter and the Chamber of Secrets").AddRow(3, "Harry Potter and the Book of Evil"))
			mock.ExpectQuery(QuoteMeta(`SELECT "book_id","id","star","text" FROM "reviews" WHERE book_id IN ($1,$2,$3)`)).WithArgs(reviewArgs, reviewArgs, reviewArgs).WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "star", "text"}).AddRow(1, 1, 5, "The Boy Who Live").AddRow(2, 2, 5, "The Girl Who Kill").AddRow(3, 3, 1, "Fake Books").AddRow(4, 1, 3, "The Man With Funny Hat"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
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
		}`, &resp, addContext(graph.NewDataSource(db)))

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
			},
		}, &resp)
	})

	t.Run("find books + reviews filter by star", func(t *testing.T) {
		if mock != nil {
			reviewArgs := NewArrayIntArgs(1, 2, 3)
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(1, "Harry Potter and the Sorcerer's Stone").AddRow(2, "Harry Potter and the Chamber of Secrets").AddRow(3, "Harry Potter and the Book of Evil"))
			mock.ExpectQuery(QuoteMeta(`SELECT "book_id","id","star","text" FROM "reviews" WHERE book_id IN ($1,$2,$3) AND star >= $4`)).WithArgs(reviewArgs, reviewArgs, reviewArgs, 3).WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "star", "text"}).AddRow(1, 1, 5, "The Boy Who Live").AddRow(2, 2, 5, "The Girl Who Kill").AddRow(4, 1, 3, "The Man With Funny Hat"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books {
				id
				title
				reviews(filter: { star: { min: 3}}) {
					id
					star
					text
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

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
			},
		}, &resp)
	})

	t.Run("find books filter by title and review.star", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT DISTINCT books.id,books.title FROM "books" JOIN reviews ON books.id = reviews.book_id WHERE books.title LIKE $1 AND reviews.star >= $2 LIMIT 10`)).WithArgs("%Harry Potter%", 3).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(1, "Harry Potter and the Sorcerer's Stone").AddRow(2, "Harry Potter and the Chamber of Secrets"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books(filter: {title: {op: LIKE, value: "%Harry Potter%"}, star: {min: 3}}) {
				id
				title
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

		JsonMatch(t, &respType{
			Books: []model.Book{
				{
					ID:    1,
					Title: "Harry Potter and the Sorcerer's Stone",
				},
				{
					ID:    2,
					Title: "Harry Potter and the Chamber of Secrets",
				},
			},
		}, &resp)
	})

	t.Run("find books concurrent", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(1, "Harry Potter and the Sorcerer's Stone").AddRow(2, "Harry Potter and the Chamber of Secrets").AddRow(3, "Harry Potter and the Book of Evil"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}

		ctx := addContext(graph.NewDataSource(db))
		var wg sync.WaitGroup
		test := func() {
			defer wg.Done()

			var resp respType
			c.MustPost(`{
			books {
				id
				title
			}
		}`, &resp, ctx)

			JsonMatch(t, &respType{
				Books: []model.Book{
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
				},
			}, &resp)
		}

		wg.Add(2)
		go test()
		go test()

		wg.Wait()
	})

	t.Run("create book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT "id" FROM "authors" WHERE name = $1 LIMIT 1`)).WithArgs("J.K. Rowling").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO "books" ("title","author_id") VALUES ($1,$2) RETURNING "id"`)).WithArgs("Harry Potter and the Unknown", 1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(4))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1)`)).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling"))
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
		c.MustPost(`mutation {
			createBook(input: {
				title: "Harry Potter and the Unknown"
				authorName: "J.K. Rowling"
			}) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			CreateBook: CreateBook{
				ID:    4,
				Title: "Harry Potter and the Unknown",
				Author: model.Author{
					ID:   1,
					Name: "J.K. Rowling",
				},
			},
		}, &resp)
	})

	t.Run("create duplicate book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT "id" FROM "authors" WHERE name = $1 LIMIT 1`)).WithArgs("J.K. Rowling").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO "books" ("title","author_id") VALUES ($1,$2) RETURNING "id"`)).WithArgs("Harry Potter and the Unknown", 1).WillReturnError(errors.New(`ERROR: duplicate key value violates unique constraint "books_title_key" (SQLSTATE 23505)`))
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
				authorName: "J.K. Rowling"
			}) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `duplicate key books.title \"Harry Potter and the Unknown\"`)
	})

	t.Run("update book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE id = $1 LIMIT 1`)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(4, "Harry Potter and the Unknown", 2))
			mock.ExpectQuery(QuoteMeta(`SELECT "id" FROM "authors" WHERE name = $1 LIMIT 1`)).WithArgs("Lord Voldermort").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`UPDATE "books" SET "title"=$1,"author_id"=$2 WHERE "id" = $3`)).WithArgs("Harry Potter and the Evil Book", 2, 4).WillReturnResult(driver.RowsAffected(1))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1)`)).WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "Lord Voldermort"))
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
		c.MustPost(`mutation {
			updateBook(input: {
				id: 4
				title: "Harry Potter and the Evil Book"
				authorName: "Lord Voldermort"
			}) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			UpdateBook: UpdateBook{
				ID:    4,
				Title: "Harry Potter and the Evil Book",
				Author: model.Author{
					ID:   2,
					Name: "Lord Voldermort",
				},
			},
		}, &resp)
	})

	t.Run("find update books", func(t *testing.T) {
		if mock != nil {
			authorArgs := NewArrayIntArgs(1, 2)
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title,books.author_id FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2).AddRow(4, "Harry Potter and the Evil Book", 2))
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1,$2)`)).WithArgs(authorArgs, authorArgs).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

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
				{
					ID:    4,
					Title: "Harry Potter and the Evil Book",
					Author: &model.Author{
						ID:   2,
						Name: "Lord Voldermort",
					},
				},
			},
		}, &resp)
	})

	t.Run("update book duplicate", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE id = $1 LIMIT 1`)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(4, "Harry Potter and the Unknown", 2))
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`UPDATE "books" SET "title"=$1 WHERE "id" = $2`)).WithArgs("Harry Potter and the Sorcerer's Stone", 4).WillReturnError(errors.New(`ERROR: duplicate key value violates unique constraint "books_title_key" (SQLSTATE 23505)`))
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
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `duplicate key books.title \"Harry Potter and the Sorcerer's Stone\"`)
	})

	t.Run("update unknown book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE id = $1 LIMIT 1`)).WithArgs(999).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}))
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
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `book with id '999' not exist`)
	})

	t.Run("delete book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE id = $1 LIMIT 1`)).WithArgs(4).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(4, "Harry Potter and the Evil Book", 2))
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`DELETE FROM "books" WHERE "books"."id" = $1`)).WithArgs(4).WillReturnResult(driver.RowsAffected(1))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1)`)).WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type DeleteBook struct {
			ID     int
			Title  string
			Author model.Author
		}
		type respType struct {
			DeleteBook DeleteBook
		}
		var resp respType
		c.MustPost(`mutation {
			deleteBook(id: 4) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		JsonMatch(t, &respType{
			DeleteBook: DeleteBook{
				ID:    4,
				Title: "Harry Potter and the Evil Book",
				Author: model.Author{
					ID:   2,
					Name: "Lord Voldermort",
				},
			},
		}, &resp)
	})

	t.Run("find delete books", func(t *testing.T) {
		if mock != nil {
			authorArgs := NewArrayIntArgs(1, 2)
			mock.ExpectQuery(QuoteMeta(`SELECT books.id,books.title,books.author_id FROM "books" LIMIT 10`)).WithArgs(NoArgs...).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}).AddRow(1, "Harry Potter and the Sorcerer's Stone", 1).AddRow(2, "Harry Potter and the Chamber of Secrets", 1).AddRow(3, "Harry Potter and the Book of Evil", 2))
			mock.ExpectQuery(QuoteMeta(`SELECT "id","name" FROM "authors" WHERE id IN ($1,$2)`)).WithArgs(authorArgs, authorArgs).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "J.K. Rowling").AddRow(2, "Lord Voldermort"))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type respType struct {
			Books []model.Book
		}
		var resp respType
		c.MustPost(`{
			books {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))

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

	t.Run("delete unknown book", func(t *testing.T) {
		if mock != nil {
			mock.ExpectQuery(QuoteMeta(`SELECT * FROM "books" WHERE id = $1 LIMIT 1`)).WithArgs(999).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author_id"}))
		}
		defer func() {
			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		}()

		type DeleteBook struct {
			ID     int
			Title  string
			Author model.Author
		}
		type respType struct {
			DeleteBook DeleteBook
		}
		var resp respType
		err := c.Post(`mutation {
			deleteBook(id: 999) {
				id
				title
				author {
					id
					name
				}
			}
		}`, &resp, addContext(graph.NewDataSource(db)))
		assert.ErrorContains(t, err, `book with id '999' not exist`)
	})
}
