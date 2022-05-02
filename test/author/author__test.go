package test_author

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog"
	"github.com/senomas/gographql/models"
	"github.com/senomas/gographql/test"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
)

func TestAuthor_DB(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

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
			Query:    graphql.NewObject(graphql.ObjectConfig{Name: "RootQuery", Fields: models.CreateFields(models.BookQueries)}),
			Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: models.CreateFields(models.AuthorMutations, models.ReviewMutations, models.BookMutations)}),
		}
		var schema graphql.Schema
		if s, err := graphql.NewSchema(schemaConfig); err != nil {
			log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
		} else {
			schema = s
		}

		eval, evalFailed := test.GqlTest(t, schema, models.NewContext(sqlDB))

		t.Run("create author", func(t *testing.T) {
			eval(`mutation {
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
		})

		t.Run("update author", func(t *testing.T) {
			eval(`mutation {
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
		})

		t.Run("update author not found", func(t *testing.T) {
			evalFailed(`mutation {
				updateAuthor(id: 9999, name: "J.K. Rowling") {
					name
				}
			}`, "affected rows 0")
		})
	}
}
