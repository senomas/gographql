package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
)

type ContextKey string

func (c ContextKey) String() string {
	return "github.com/senomas/gographql:" + string(c)
}

const ContextKeyDB = ContextKey("db")

func NewContext(sqlDB *sql.DB) context.Context {
	return context.WithValue(context.Background(), ContextKeyDB, sqlDB)
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
