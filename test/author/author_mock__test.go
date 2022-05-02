package test_author

import (
	"database/sql/driver"
	"encoding/json"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/graphql-go/graphql"
	"github.com/senomas/gographql/models"
	"github.com/stretchr/testify/assert"
)

func TestAuthor(t *testing.T) {
	if sqlDB, mock, err := sqlmock.New(); err != nil {
		t.Fatal("init SQLMock Error", err)
	} else {
		defer sqlDB.Close()

		schemaConfig := graphql.SchemaConfig{
			Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: models.CreateFields(sqlDB, models.BookQueries)}),
			Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: models.CreateFields(sqlDB, models.AuthorMutations, models.ReviewMutations, models.BookMutations)}),
		}
		var schema graphql.Schema
		if s, err := graphql.NewSchema(schemaConfig); err != nil {
			log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
		} else {
			schema = s
		}

		t.Run("create author", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(QuoteMeta(`INSERT INTO authors (name) VALUES ($1) RETURNING id`)).WithArgs("Lord Voldemort").WillReturnRows(sqlmock.NewRows(
				[]string{"id"}).
				AddRow(1))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT id, name FROM authors WHERE id = $1`)).WithArgs(1).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "name"}).
				AddRow(1, "Lord Voldemort"))

			query := `
			mutation {
				createAuthor(name: "Lord Voldemort") {
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
			"name": "Lord Voldemort"
		}
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("update author", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectExec(QuoteMeta(`UPDATE authors SET name = $2 WHERE id = $1`)).WithArgs(1, "J.K. Rowling").WillReturnResult(driver.RowsAffected(1))
			mock.ExpectCommit()
			mock.ExpectQuery(QuoteMeta(`SELECT id, name FROM authors WHERE id = $1`)).WithArgs(1).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "name"}).
				AddRow(1, "J.K. Rowling"))

			query := `
			mutation {
				updateAuthor(id: 1, name: "J.K. Rowling") {
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
		"updateAuthor": {
			"name": "J.K. Rowling"
		}
	}
}`, string(rJSON))

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
