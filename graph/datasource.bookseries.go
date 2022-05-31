package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
	"gorm.io/gorm"
)

func (ds *DataSource) BookSeries(ctx context.Context, offset *int, limit *int, filter *model.BookSeriesFilter) (*model.BookSeriesList, error) {
	needCount := false
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		switch f.Name {
		case "count":
			needCount = true
		case "list":
			for _, f := range graphql.CollectFields(graphql.GetOperationContext(ctx), f.SelectionSet, nil) {
				switch f.Name {
            case "books":
				default:
					fields = append(fields, fmt.Sprintf(`"book_series"."%s"`, f.Name))
				}
			}
		}
	}
	var scopeFn = func(offset *int, limit *int) func(tx *gorm.DB) *gorm.DB {
		return func(tx *gorm.DB) *gorm.DB {
			tx.Model(&model.BookSeries{})
			if filter != nil {
				if filter.ID != nil {
					tx.Where("book_series.id = ?", filter.ID)
				}
				if filter.Title != nil {
					FilterText(filter.Title, tx, "book_series.title")
				}
			}
			if offset != nil {
				tx.Offset(*offset)
			}
			if limit != nil {
				tx.Limit(*limit)
			}
			return tx
		}
	}
	tx := ds.DB.Session(&gorm.Session{DryRun: true}).Select(fields).Scopes(scopeFn(offset, limit)).Find(&model.Book{})
	group := tx.Statement.SQL.String()
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	var queryFn = func(keys []*BatchLoaderKey) *dataloader.Result {
		var bookSeries []*model.BookSeries
		var count int64
		var result *gorm.DB
		if needCount {
			result = ds.DB.Scopes(scopeFn(nil, nil)).Count(&count)
			if result.Error != nil {
				return &dataloader.Result{
					Error: result.Error,
				}
			}
			if count == 0 {
				return &dataloader.Result{
					Data: &model.BookSeriesList{List: []*model.BookSeries{}, Count: int(count)},
				}
			}
		}
		result = ds.DB.Select(fields).Scopes(scopeFn(offset, limit)).Find(&bookSeries)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: &model.BookSeriesList{List: bookSeries, Count: int(count)},
		}
	}
	data, err := ds.BatchLoad(ctx, &group, key, nil, nil, queryFn, nil)
	if data != nil {
		return data.(*model.BookSeriesList), err
	}
	return nil, err
}
