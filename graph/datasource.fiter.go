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
	case model.FilterTextOpEq:
		tx.Where(fmt.Sprintf("%s = ?", field), filter.Value)
	}
}
