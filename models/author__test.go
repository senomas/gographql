package models

import (
	"database/sql/driver"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
)

func TestAuthor(t *testing.T) {
	if sqlDB, mock, err := sqlmock.New(); err != nil {
		t.Fatal("init SQLMock Error", err)
	} else {
		defer sqlDB.Close()

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

		testQL, _ := QLTest(t, schema, NewContext(sqlDB))

		t.Run("create author", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO authors (name) VALUES ($1) RETURNING id`)).WithArgs("Lord Voldemort").WillReturnRows(sqlmock.NewRows(
				[]string{"id"}).
				AddRow(1))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT id, name FROM authors WHERE id = $1`)).WithArgs(1).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "name"}).
				AddRow(1, "Lord Voldemort"))

			testQL(`mutation {
				createAuthor(name: "Lord Voldemort") {
					id
					name
				}
			}`, `{
				"data": {
					"createAuthor": {
						"id": 1,
						"name": "Lord Voldemort"
					}
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("update author", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`UPDATE authors SET name = $2 WHERE id = $1`)).WithArgs(1, "J.K. Rowling").WillReturnResult(driver.RowsAffected(1))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT id, name FROM authors WHERE id = $1`)).WithArgs(1).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "name"}).
				AddRow(1, "J.K. Rowling"))

			testQL(`mutation {
				updateAuthor(id: 1, name: "J.K. Rowling") {
					name
				}
			}`, `{
				"data": {
					"updateAuthor": {
						"name": "J.K. Rowling"
					}
				}
			}`)

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}