package model

import (
	"fmt"
	"io"
	"strconv"
)

type Author struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique"`
}

type AuthorFilter struct {
	ID   *int        `json:"id"`
	Name *FilterText `json:"name"`
}

type AuthorList struct {
	List  []*Author `json:"list"`
	Count int       `json:"count"`
}

type Book struct {
	ID       int       `json:"id" gorm:"primaryKey"`
	Title    string    `json:"title" gorm:"unique"`
	AuthorID int       `json:"-"`
	Author   *Author   `json:"author"`
	Reviews  []*Review `json:"reviews"`
}

type BookFilter struct {
	ID         *int            `json:"id"`
	Title      *FilterText     `json:"title"`
	AuthorName *FilterText     `json:"author_name"`
	Star       *FilterIntRange `json:"star"`
}

type BookList struct {
	List  []*Book `json:"list"`
	Count int     `json:"count"`
}

type FilterIntRange struct {
	Min *int `json:"min"`
	Max *int `json:"max"`
}

type FilterText struct {
	Op    FilterTextOp `json:"op"`
	Value string       `json:"value"`
}

type NewAuthor struct {
	Name string `json:"name"`
}

type NewBook struct {
	Title      string `json:"title"`
	AuthorName string `json:"authorName"`
}

type NewReview struct {
	BookID int    `json:"book_id"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
}

type Review struct {
	ID     int    `json:"id" gorm:"primaryKey"`
	BookID int    `json:"-"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
	Book   *Book  `json:"book"`
}

type ReviewFilter struct {
	Star *FilterIntRange `json:"star"`
}

type UpdateBook struct {
	ID         int     `json:"id"`
	Title      *string `json:"title"`
	AuthorName *string `json:"author_name"`
}

type FilterTextOp string

const (
	FilterTextOpLike FilterTextOp = "LIKE"
	FilterTextOpEq   FilterTextOp = "EQ"
)

var AllFilterTextOp = []FilterTextOp{
	FilterTextOpLike,
	FilterTextOpEq,
}

func (e FilterTextOp) IsValid() bool {
	switch e {
	case FilterTextOpLike, FilterTextOpEq:
		return true
	}
	return false
}

func (e FilterTextOp) String() string {
	return string(e)
}

func (e *FilterTextOp) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = FilterTextOp(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid FilterTextOp", str)
	}
	return nil
}

func (e FilterTextOp) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
