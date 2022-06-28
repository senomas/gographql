package graph_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/senomas/gographql/graph"
	"github.com/senomas/gographql/graph/generated"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var NoArgs = []driver.Value{}

type GraphqlError struct {
	Message    string
	Path       []string
	Extensions map[string]interface{}
}

func addContext(ds *graph.DataSource) client.Option {
	return func(bd *client.Request) {
		ctx := context.WithValue(context.TODO(), graph.Context_DataSource, ds)
		bd.HTTP = bd.HTTP.WithContext(ctx)
	}
}

func JsonMatch(t *testing.T, expected interface{}, resp interface{}, msg ...string) {
	rJSON, _ := json.MarshalIndent(resp, "", "\t")
	eJSON, _ := json.MarshalIndent(expected, "", "\t")

	assert.Equal(t, string(eJSON), string(rJSON), msg)
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

func SetupTest() (generated.Config, *handler.Server, *client.Client) {
	cfg := generated.Config{Resolvers: &graph.Resolver{}}
	h := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))
	c := client.New(h)
	return cfg, h, c
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
					Colorful:                  false,
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
				Colorful:                  false,
			},
		)
	}

	if dsnPostgre != nil {
		if db, err := gorm.Open(postgres.New(postgres.Config{DSN: *dsnPostgre}), &gorm.Config{Logger: gormLogger}); err != nil {
			return nil, nil, nil, err
		} else {
			if err := graph.Migrate(db); err != nil {
				return nil, nil, nil, err
			}

			tx := db.Begin()
			if err := Populate(tx); err != nil {
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
			return sqlDB, db, mock, nil
		}
	}
}

func Populate(tx *gorm.DB) error {
	return graph.Populate(tx)
}

func QuoteMeta(r string) string {
	r = strings.Join(strings.Fields(r), " ")
	r = strings.ReplaceAll(r, "( ", "(")
	r = strings.ReplaceAll(r, " )", ")")
	return "^" + regexp.QuoteMeta(r) + "$"
}
