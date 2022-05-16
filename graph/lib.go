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
		db.Migrator().DropTable(&model.Author{}, &model.Book{}, &model.Review{}, "book_authors")
		if err := db.AutoMigrate(&model.Author{}, &model.Book{}, &model.Review{}); err != nil {
			return nil, nil, err
		}

		if err := Populate(db); err != nil {
			return nil, nil, err
		}

		if sqlDB, err := db.DB(); err != nil {
			return nil, nil, err
		} else {
			return sqlDB, db, nil
		}
	}
}

func Populate(db *gorm.DB) error {
	tx := db.Begin()

	var author = model.Author{
		Name: "J.K. Rowling",
	}
	if result := tx.Create(&author); result.Error != nil {
		return result.Error
	}
	jkRowling := author
	author = model.Author{
		Name: "Lord Voldermort",
	}
	if result := tx.Create(&author); result.Error != nil {
		return result.Error
	}
	lordVoldermort := author
	author = model.Author{
		Name: "Salazar Slitherin",
	}
	if result := tx.Create(&author); result.Error != nil {
		return result.Error
	}
	salazarSlitherin := author
	author = model.Author{
		Name: "Albus Dumbledore",
	}
	if result := tx.Create(&author); result.Error != nil {
		return result.Error
	}

	var book = model.Book{
		Title:   "Harry Potter and the Sorcerer's Stone",
		Authors: []*model.Author{&jkRowling},
	}
	if result := tx.Create(&book); result.Error != nil {
		return result.Error
	}
	book = model.Book{
		Title:   "Harry Potter and the Chamber of Secrets",
		Authors: []*model.Author{&jkRowling},
	}
	if result := tx.Create(&book); result.Error != nil {
		return result.Error
	}
	book = model.Book{
		Title:   "Harry Potter and the Book of Evil",
		Authors: []*model.Author{&lordVoldermort},
	}
	if result := tx.Create(&book); result.Error != nil {
		return result.Error
	}
	book = model.Book{
		Title:   "Harry Potter and the Snake Dictionary",
		Authors: []*model.Author{&lordVoldermort, &salazarSlitherin},
	}
	if result := tx.Create(&book); result.Error != nil {
		return result.Error
	}

	review := model.Review{
		BookID: 1,
		Star:   5,
		Text:   "The Boy Who Live",
	}
	if result := tx.Create(&review); result.Error != nil {
		return result.Error
	}

	review = model.Review{
		BookID: 2,
		Star:   5,
		Text:   "The Girl Who Kill",
	}
	if result := tx.Create(&review); result.Error != nil {
		return result.Error
	}

	review = model.Review{
		BookID: 3,
		Star:   1,
		Text:   "Fake Books",
	}
	if result := tx.Create(&review); result.Error != nil {
		return result.Error
	}

	review = model.Review{
		BookID: 1,
		Star:   3,
		Text:   "The Man With Funny Hat",
	}
	if result := tx.Create(&review); result.Error != nil {
		return result.Error
	}

	tx.Commit()
	return nil
}

func Directive_Gorm(ctx context.Context, obj interface{}, next graphql.Resolver, tag *string, ref *string) (interface{}, error) {
	return next(ctx)
}
