package graph

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/senomas/gographql/graph/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Setup() (*sql.DB, *gorm.DB, error) {
	var dsnPostgre string
	var gormLogger logger.Interface
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if pair[0] == "DB_POSTGRES" {
			dsnPostgre = pair[1]
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

	if dsnPostgre == "" {
		dsnPostgre = "host=localhost user=demo password=password dbname=demo port=5432 sslmode=disable TimeZone=Asia/Jakarta"
	}

	if db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsnPostgre}), &gorm.Config{Logger: gormLogger}); err != nil {
		return nil, nil, err
	} else {
		db.Migrator().DropTable(&model.Author{}, &model.Book{}, &model.Review{})
		if err := db.AutoMigrate(&model.Author{}, &model.Book{}, &model.Review{}); err != nil {
			return nil, nil, err
		}

		tx := db.Begin()

		var author = model.Author{
			Name: "J.K. Rowling",
		}
		if result := tx.Create(&author); result.Error != nil {
			return nil, nil, err
		}
		author = model.Author{
			Name: "Lord Voldermort",
		}
		if result := tx.Create(&author); result.Error != nil {
			return nil, nil, err
		}

		var book = model.Book{
			Title:    "Harry Potter and the Sorcerer's Stone",
			AuthorID: 1,
		}
		if result := tx.Create(&book); result.Error != nil {
			return nil, nil, err
		}
		book = model.Book{
			Title:    "Harry Potter and the Chamber of Secrets",
			AuthorID: 1,
		}
		if result := tx.Create(&book); result.Error != nil {
			return nil, nil, err
		}
		book = model.Book{
			Title:    "Harry Potter and the Book of Evil",
			AuthorID: 2,
		}
		if result := tx.Create(&book); result.Error != nil {
			return nil, nil, err
		}

		review := model.Review{
			BookID: 1,
			Star:   5,
			Text:   "The Boy Who Live",
		}
		if result := tx.Create(&review); result.Error != nil {
			return nil, nil, err
		}

		review = model.Review{
			BookID: 2,
			Star:   5,
			Text:   "The Girl Who Kill",
		}
		if result := tx.Create(&review); result.Error != nil {
			return nil, nil, err
		}

		review = model.Review{
			BookID: 3,
			Star:   1,
			Text:   "Fake Books",
		}
		if result := tx.Create(&review); result.Error != nil {
			return nil, nil, err
		}

		review = model.Review{
			BookID: 1,
			Star:   3,
			Text:   "The Man With Funny Hat",
		}
		if result := tx.Create(&review); result.Error != nil {
			return nil, nil, err
		}

		tx.Commit()

		if sqlDB, err := db.DB(); err != nil {
			return nil, nil, err
		} else {
			return sqlDB, db, nil
		}
	}
}

func Directive_Gorm(ctx context.Context, obj interface{}, next graphql.Resolver, tag *string, ref *string) (interface{}, error) {
	return next(ctx)
}
