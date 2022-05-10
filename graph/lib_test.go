package graph_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/senomas/gographql/graph/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var NoArgs = []driver.Value{}

func JsonMatch(t *testing.T, expected interface{}, resp interface{}) {
	rJSON, _ := json.MarshalIndent(resp, "", "\t")
	eJSON, _ := json.MarshalIndent(expected, "", "\t")

	assert.Equal(t, string(eJSON), string(rJSON))
}

func Setup() (*sql.DB, *gorm.DB, sqlmock.Sqlmock, error) {
	if sqlDB, mock, err := sqlmock.New(); err != nil {
		return sqlDB, nil, mock, err
	} else {
		if db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{}); err != nil {
			return sqlDB, db, mock, err
		} else {
			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).WithArgs("authors", "BASE TABLE").WillReturnRows(sqlmock.NewRows(
				[]string{"count"}).
				AddRow(0))

			mock.ExpectExec(QuoteMeta(`CREATE TABLE "authors" ("id" bigserial,"name" text UNIQUE,PRIMARY KEY ("id"))`)).WithArgs(NoArgs...).WillReturnResult(driver.RowsAffected(1))

			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).WithArgs("books", "BASE TABLE").WillReturnRows(sqlmock.NewRows(
				[]string{"count"}).
				AddRow(0))

			mock.ExpectExec(QuoteMeta(`CREATE TABLE "books" ("id" bigserial,"title" text UNIQUE,"author_id" bigint,PRIMARY KEY ("id"),CONSTRAINT "fk_books_author" FOREIGN KEY ("author_id") REFERENCES "authors"("id"))`)).WithArgs(NoArgs...).WillReturnResult(driver.RowsAffected(1))

			db.AutoMigrate(&model.Author{}, &model.Book{})

			return sqlDB, db, mock, mock.ExpectationsWereMet()
		}
	}
}

func QuoteMeta(r string) string {
	return "^" + regexp.QuoteMeta(r) + "$"
}
