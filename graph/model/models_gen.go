// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

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
	ID      int       `json:"id" gorm:"primaryKey"`
	Title   string    `json:"title" gorm:"unique"`
	Authors []*Author `json:"authors" gorm:"many2many:book_authors;constraint:OnDelete:CASCADE"`
	Reviews []*Review `json:"reviews" gorm:"constraint:OnDelete:CASCADE"`
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
	Title       string   `json:"title"`
	AuthorsName []string `json:"authors_name"`
}

type NewReview struct {
	BookID int    `json:"book_id"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
}

type Review struct {
	ID     int    `json:"id" gorm:"primaryKey"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
	Book   *Book  `json:"book"`
	BookID int    `json:"-"`
}

type ReviewFilter struct {
	Star *FilterIntRange `json:"star"`
}

type UpdateBook struct {
	ID          int      `json:"id"`
	Title       *string  `json:"title"`
	AuthorsName []string `json:"authors_name"`
}

type FilterTextOp string

const (
	FilterTextOpLike    FilterTextOp = "LIKE"
	FilterTextOpEq      FilterTextOp = "EQ"
	FilterTextOpNotLike FilterTextOp = "NOT_LIKE"
	FilterTextOpNotEq   FilterTextOp = "NOT_EQ"
)

var AllFilterTextOp = []FilterTextOp{
	FilterTextOpLike,
	FilterTextOpEq,
	FilterTextOpNotLike,
	FilterTextOpNotEq,
}

func (e FilterTextOp) IsValid() bool {
	switch e {
	case FilterTextOpLike, FilterTextOpEq, FilterTextOpNotLike, FilterTextOpNotEq:
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
