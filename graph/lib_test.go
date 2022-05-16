package graph_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/senomas/gographql/graph/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var NoArgs = []driver.Value{}

func JsonMatch(t *testing.T, expected interface{}, resp interface{}) {
	rJSON, _ := json.MarshalIndent(resp, "", "\t")
	eJSON, _ := json.MarshalIndent(expected, "", "\t")

	assert.Equal(t, string(eJSON), string(rJSON))
}

type ArrayIntArgs struct {
	value map[int64]bool
}

func NewArrayIntArgs(args ...int64) *ArrayIntArgs {
	v := ArrayIntArgs{value: make(map[int64]bool)}
	for _, a := range args {
		v.value[a] = true
	}
	return &v
}

func (m *ArrayIntArgs) Match(v driver.Value) bool {
	for i, unused := range m.value {
		if i == v.(int64) {
			if !unused {
				return false
			}
			m.value[i] = true
			return true
		}
	}
	return false
}

func Setup() (*sql.DB, *gorm.DB, sqlmock.Sqlmock, error) {
	var dsnPostgre *string
	var gormLogger logger.Interface
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if pair[0] == "TEST_DB_POSTGRES" {
			dsnPostgre = &pair[1]
		} else if pair[0] == "LOGGER" && pair[1] != "" {
			gormLogger = logger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold:             time.Second,
					LogLevel:                  logger.Info,
					IgnoreRecordNotFoundError: true,
					Colorful:                  true,
				},
			)
		}
	}
	if gormLogger == nil {
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	}

	if dsnPostgre != nil {
		if db, err := gorm.Open(postgres.New(postgres.Config{DSN: *dsnPostgre}), &gorm.Config{Logger: gormLogger}); err != nil {
			return nil, nil, nil, err
		} else {
			db.Migrator().DropTable(&model.Author{}, &model.Book{}, &model.Review{})
			if err := db.AutoMigrate(&model.Author{}, &model.Book{}, &model.Review{}); err != nil {
				return nil, nil, nil, err
			}

			tx := db.Begin()

			var author = model.Author{
				Name: "J.K. Rowling",
			}
			if result := tx.Create(&author); result.Error != nil {
				return nil, nil, nil, err
			}
			author = model.Author{
				Name: "Lord Voldermort",
			}
			if result := tx.Create(&author); result.Error != nil {
				return nil, nil, nil, err
			}

			var book = model.Book{
				Title:    "Harry Potter and the Sorcerer's Stone",
				AuthorID: 1,
			}
			if result := tx.Create(&book); result.Error != nil {
				return nil, nil, nil, err
			}
			book = model.Book{
				Title:    "Harry Potter and the Chamber of Secrets",
				AuthorID: 1,
			}
			if result := tx.Create(&book); result.Error != nil {
				return nil, nil, nil, err
			}
			book = model.Book{
				Title:    "Harry Potter and the Book of Evil",
				AuthorID: 2,
			}
			if result := tx.Create(&book); result.Error != nil {
				return nil, nil, nil, err
			}

			review := model.Review{
				BookID: 1,
				Star:   5,
				Text:   "The Boy Who Live",
			}
			if result := tx.Create(&review); result.Error != nil {
				return nil, nil, nil, err
			}

			review = model.Review{
				BookID: 2,
				Star:   5,
				Text:   "The Girl Who Kill",
			}
			if result := tx.Create(&review); result.Error != nil {
				return nil, nil, nil, err
			}

			review = model.Review{
				BookID: 3,
				Star:   1,
				Text:   "Fake Books",
			}
			if result := tx.Create(&review); result.Error != nil {
				return nil, nil, nil, err
			}

			review = model.Review{
				BookID: 1,
				Star:   3,
				Text:   "The Man With Funny Hat",
			}
			if result := tx.Create(&review); result.Error != nil {
				return nil, nil, nil, err
			}

			tx.Commit()

			if sqlDB, err := db.DB(); err != nil {
				return nil, nil, nil, err
			} else {
				return sqlDB, db, nil, nil
			}
		}
	}
	if sqlDB, mock, err := sqlmock.New(); err != nil {
		return sqlDB, nil, mock, err
	} else {
		if db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: gormLogger}); err != nil {
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

			mock.ExpectQuery(QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1 AND table_type = $2`)).WithArgs("reviews", "BASE TABLE").WillReturnRows(sqlmock.NewRows(
				[]string{"count"}).
				AddRow(0))

			mock.ExpectExec(QuoteMeta(`CREATE TABLE "reviews" ("id" bigserial,"star" bigint,"text" text,"book_id" bigint,PRIMARY KEY ("id"),CONSTRAINT "fk_books_reviews" FOREIGN KEY ("book_id") REFERENCES "books"("id"))`)).WithArgs(NoArgs...).WillReturnResult(driver.RowsAffected(1))

			db.AutoMigrate(&model.Author{}, &model.Book{}, &model.Review{})

			return sqlDB, db, mock, mock.ExpectationsWereMet()
		}
	}
}

func QuoteMeta(r string) string {
	return "^" + regexp.QuoteMeta(r) + "$"
}
