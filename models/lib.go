package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/graphql-go/graphql"
	"github.com/senomas/gographql/data"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ContextKey string

func (c ContextKey) String() string {
	return "github.com/senomas/gographql:" + string(c)
}

const ContextKeyDB = ContextKey("db")
const ContextKeyLoader = ContextKey("loader")

var NoArgs = []driver.Value{}

func NewContext(sqlDB *sql.DB) context.Context {
	if db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{}); err != nil {
		fmt.Printf("database error, %v\n", err)
	} else {
		loader := data.NewLoader(sqlDB)
		return context.WithValue(context.WithValue(context.Background(), ContextKeyDB, db), ContextKeyLoader,
			loader)
	}
	return context.Background()
}

func CreateFields(fns ...func(fields graphql.Fields) graphql.Fields) graphql.Fields {
	fields := graphql.Fields{}
	for _, fn := range fns {
		fields = fn(fields)
	}
	return fields
}

func SetupDevDatabase(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS reviews CASCADE"); err != nil {
		return fmt.Errorf("failed to drop reviews, err %v", err)
	}

	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS books CASCADE"); err != nil {
		return fmt.Errorf("failed to drop books, err %v", err)
	}

	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS authors CASCADE"); err != nil {
		return fmt.Errorf("failed to drop authors, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE TABLE authors (id bigserial, name text, PRIMARY KEY (id))"); err != nil {
		return fmt.Errorf("failed to create author, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_authors_name ON authors (name)"); err != nil {
		return fmt.Errorf("failed to create idx_authors_name, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE TABLE books (id bigserial, title text, author_id bigint, PRIMARY KEY (id), CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id))"); err != nil {
		return fmt.Errorf("failed to create authors, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_books_title ON books (title)"); err != nil {
		return fmt.Errorf("failed to create idx_books_title, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE TABLE reviews (id bigserial, book_id bigint, star smallint, body text, PRIMARY KEY (id), CONSTRAINT fk_book FOREIGN KEY (book_id) REFERENCES books(id))"); err != nil {
		return fmt.Errorf("failed to create books, err %v", err)
	}
	return nil
}

func QuoteMeta(r string) string {
	return "^" + regexp.QuoteMeta(r) + "$"
}

func SetupGorm(t *testing.T, mock sqlmock.Sqlmock, sqlDB *sql.DB) (graphql.Schema, context.Context) {
	mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).WithArgs("authors", "BASE TABLE").WillReturnRows(sqlmock.NewRows(
		[]string{"count"}).
		AddRow(0))

	mock.ExpectExec(QuoteMeta(`CREATE TABLE "authors" ("id" bigserial,"name" text,PRIMARY KEY ("id"))`)).WithArgs(NoArgs...).WillReturnResult(driver.RowsAffected(1))

	mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).WithArgs("reviews", "BASE TABLE").WillReturnRows(sqlmock.NewRows(
		[]string{"count"}).
		AddRow(0))

	mock.ExpectExec(QuoteMeta(`CREATE TABLE "reviews" ("id" bigserial,"star" bigint,"body" text,"book_id" bigint,PRIMARY KEY ("id"),CONSTRAINT "fk_books_reviews" FOREIGN KEY ("book_id") REFERENCES "books"("id"))`)).WithArgs(NoArgs...).WillReturnResult(driver.RowsAffected(1))

	mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).WithArgs("books", "BASE TABLE").WillReturnRows(sqlmock.NewRows(
		[]string{"count"}).
		AddRow(0))

	mock.ExpectExec(QuoteMeta(`CREATE TABLE "books" ("id" bigserial,"title" text,"author_id" bigint,PRIMARY KEY ("id"),CONSTRAINT "fk_books_author" FOREIGN KEY ("author_id") REFERENCES "authors"("id"))`)).WithArgs(NoArgs...).WillReturnResult(driver.RowsAffected(1))

	ctx := NewContext(sqlDB)

	if err := ctx.Value(ContextKeyDB).(*gorm.DB).AutoMigrate(&data.Author{}, &data.Review{}, &data.Book{}); err != nil {
		t.Fatalf("auto migrate error, %v\n", err)
	}

	schemaConfig := graphql.SchemaConfig{
		Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: CreateFields(BookQueries)}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: CreateFields(AuthorMutations, ReviewMutations, BookMutations)}),
	}
	var schema graphql.Schema
	if s, err := graphql.NewSchema(schemaConfig); err != nil {
		log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
	} else {
		schema = s
	}
	return schema, ctx
}

func QLTest(t *testing.T, schema graphql.Schema, ctx context.Context) (func(query string, str string) *graphql.Result, func(query string, str string) *graphql.Result) {
	return func(query string, str string) *graphql.Result {
			params := graphql.Params{
				Schema:        schema,
				RequestString: query,
				RootObject:    make(map[string]interface{}),
				Context:       ctx,
			}
			r := graphql.Do(params)

			rJSON, _ := json.MarshalIndent(r, "", "\t")

			v := make(map[string]interface{})
			json.Unmarshal([]byte(str), &v)
			eJSON, _ := json.MarshalIndent(v, "", "\t")

			assert.Equal(t, string(eJSON), string(rJSON))

			return r
		}, func(query string, str string) *graphql.Result {
			params := graphql.Params{
				Schema:        schema,
				RequestString: query,
				RootObject:    make(map[string]interface{}),
				Context:       ctx,
			}
			r := graphql.Do(params)

			assert.Equal(t, len(r.Errors), 1)
			assert.ErrorContains(t, r.Errors[0], str)

			return r
		}
}
