package graph

import (
	"fmt"

	"github.com/senomas/gographql/graph/model"
	"gorm.io/gorm"
)

func FilterText(filter *model.FilterText, tx *gorm.DB, field string) {
	switch filter.Op {
	case model.FilterTextOpLike:
		tx.Where(fmt.Sprintf("%s LIKE ?", field), filter.Value)
	case model.FilterTextOpNotLike:
		tx.Where(fmt.Sprintf("%s NOT LIKE ?", field), filter.Value)
	case model.FilterTextOpEq:
		tx.Where(fmt.Sprintf("%s = ?", field), filter.Value)
	case model.FilterTextOpNotEq:
		tx.Where(fmt.Sprintf("%s != ?", field), filter.Value)
	}
}

func FilterSubQueryText(filter *model.FilterText, tx *gorm.DB, field string) string {
	op := "IN"
	switch filter.Op {
	case model.FilterTextOpLike:
		tx.Where(fmt.Sprintf("%s LIKE ?", field), filter.Value)
	case model.FilterTextOpNotLike:
		tx.Where(fmt.Sprintf("%s LIKE ?", field), filter.Value)
		op = "NOT IN"
	case model.FilterTextOpEq:
		tx.Where(fmt.Sprintf("%s = ?", field), filter.Value)
	case model.FilterTextOpNotEq:
		tx.Where(fmt.Sprintf("%s = ?", field), filter.Value)
		op = "NOT IN"
	}
	return op
}

func FilterIntRange(filter *model.FilterIntRange, tx *gorm.DB, field string) {
	if filter.Min != nil {
		tx.Where(fmt.Sprintf("%s >= ?", field), filter.Min)
	}
	if filter.Max != nil {
		tx.Where(fmt.Sprintf("%s >= ?", field), filter.Max)
	}
}
