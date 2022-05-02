package models

import (
	"database/sql"
	"fmt"

	"github.com/graphql-go/graphql"
)

func CreateFields(db *sql.DB, fns ...func(db *sql.DB, fields graphql.Fields) graphql.Fields) graphql.Fields {
	fields := graphql.Fields{}
	for _, fn := range fns {
		fields = fn(db, fields)
	}
	return fields
}

func SetupDevDatabase(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS reviews"); err != nil {
		return fmt.Errorf("failed to drop reviews, err %v", err)
	}

	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS books"); err != nil {
		return fmt.Errorf("failed to drop books, err %v", err)
	}

	if _, err := sqlDB.Exec("DROP TABLE IF EXISTS authors"); err != nil {
		return fmt.Errorf("failed to drop authors, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS authors (id bigserial, name text, PRIMARY KEY (id))"); err != nil {
		return fmt.Errorf("failed to create author, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_authors_name ON authors (name)"); err != nil {
		return fmt.Errorf("failed to create idx_authors_name, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS books (id bigserial, title text, author_id bigint, PRIMARY KEY (id), CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES authors(id))"); err != nil {
		return fmt.Errorf("failed to create authors, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_books_title ON books (title)"); err != nil {
		return fmt.Errorf("failed to create idx_books_title, err %v", err)
	}

	if _, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS reviews (id bigserial, book_id bigint, star smallint, body text, PRIMARY KEY (id), CONSTRAINT fk_book FOREIGN KEY (book_id) REFERENCES books(id))"); err != nil {
		return fmt.Errorf("failed to create books, err %v", err)
	}
	return nil
}
