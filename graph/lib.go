package graph

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/senomas/gographql/graph/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Models = []interface{}{&model.Author{}, &model.Book{}, &model.BookSeries{}, &model.Review{}}
var RefTables = []interface{}{}

type ConfigType struct {
	Application          string
	TokenSecret          string
	HashedPasswordLength uint32
	Argon2_Time          uint32
	Argon2_Memory        uint32
	Argon2_Thread        uint8
}

var Config = ConfigType{
	Application:          "MyApp",
	TokenSecret:          "supersecure",
	HashedPasswordLength: 32,
	Argon2_Time:          3,
	Argon2_Memory:        64 * 1024,
	Argon2_Thread:        2,
}

func Of[E any](e E) *E {
	return &e
}

func ParseTime(v string) *time.Time {
	res, err := time.ParseInLocation("2006-01-02T15:04:05", v, time.Local)
	if err != nil {
		panic(fmt.Sprintf("Invalid date '%s' %#v\n", v, err))
	}
	return &res
}

func JsonStr(v interface{}) string {
	rJSON, _ := json.MarshalIndent(v, "", "\t")
	return string(rJSON)
}

func GenerateRandomString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret)
}

func Migrate(db *gorm.DB) error {
	if err := db.Migrator().DropTable(Models...); err != nil {
		return err
	}
	if err := db.Migrator().DropTable(RefTables...); err != nil {
		return err
	}
	if err := db.AutoMigrate(Models...); err != nil {
		return err
	}
	return nil
}

func Setup() (*sql.DB, *gorm.DB, error) {
	// if salt, ok := os.LookupEnv("PASSWORD_SALT"); ok {
	// 	Config.Salt = salt
	// }

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
		if err := Migrate(db); err != nil {
			return nil, nil, err
		}

		tx := db.Begin()
		if err := Populate(tx); err != nil {
			return nil, nil, err
		}
		db.Commit()

		if sqlDB, err := db.DB(); err != nil {
			return nil, nil, err
		} else {
			return sqlDB, db, nil
		}
	}
}

func Populate(tx *gorm.DB) error {
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

	var bookSeries = model.BookSeries{
		Title: "Harry Potter",
	}
	if result := tx.Create(&bookSeries); result.Error != nil {
		return result.Error
	}
	var book = model.Book{
		Title:   "Harry Potter and the Sorcerer's Stone",
		Series:  &bookSeries,
		Authors: []*model.Author{&jkRowling},
	}
	if result := tx.Create(&book); result.Error != nil {
		return result.Error
	}
	book = model.Book{
		Title:   "Harry Potter and the Chamber of Secrets",
		Series:  &bookSeries,
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
	return nil
}
