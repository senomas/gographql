package data

import (
	"database/sql"
	"fmt"

	"github.com/graph-gophers/dataloader"
)

type Loader struct {
	conn          *sql.DB
	booksLoader   *dataloader.Loader
	reviewsLoader *dataloader.Loader
}

func NewLoader(conn *sql.DB) *Loader {
	l := &Loader{conn: conn}
	l.reviewsLoader = dataloader.NewBatchedLoader(l.getReviews)
	l.booksLoader = dataloader.NewBatchedLoader(l.getBooks)
	return l
}

type DataKey struct {
	data interface{}
}

func NewDataKey(data interface{}) *DataKey {
	return &DataKey{data: data}
}

func (k *DataKey) String() string {
	return fmt.Sprintf("%#v", k.data)
}

func (k *DataKey) Raw() interface{} {
	return k.data
}
