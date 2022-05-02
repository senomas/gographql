package test_author

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog"
	"github.com/senomas/gographql/models"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
	"github.com/stretchr/testify/assert"
)

func TestAuthor_DB(t *testing.T) {
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
		})

		t.Run("update author", func(t *testing.T) {
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
		})

		t.Run("update author not found", func(t *testing.T) {
			query := `
			mutation {
				updateAuthor(id: 9999, name: "J.K. Rowling") {
					name
				}
			}`
			params := graphql.Params{Schema: schema, RequestString: query}

			r := graphql.Do(params)
			assert.Equal(t, len(r.Errors), 1)
			assert.ErrorContains(t, r.Errors[0], "affected rows 0")
		})
	}
}

func QuoteMeta(r string) string {
	return "^" + regexp.QuoteMeta(r) + "$"
}
