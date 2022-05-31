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
	"github.com/senomas/gographql/graph"
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
			if err := db.Migrator().DropTable(&model.BookSeries{}, &model.Author{}, &model.Book{}, &model.Review{}, "book_authors"); err != nil {
				return nil, nil, nil, err
			}
			if err := db.AutoMigrate(&model.BookSeries{}, &model.Author{}, &model.Book{}, &model.Review{}); err != nil {
				return nil, nil, nil, err
			}

			if err := graph.Populate(db); err != nil {
				return nil, nil, nil, err
			}

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
			return sqlDB, db, mock, nil
		}
	}
}

func QuoteMeta(r string) string {
	r = strings.Join(strings.Fields(r), " ")
	r = strings.ReplaceAll(r, "( ", "(")
	r = strings.ReplaceAll(r, " )", ")")
	return "^" + regexp.QuoteMeta(r) + "$"
}
