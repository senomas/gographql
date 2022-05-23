package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
	"gorm.io/gorm"
)

func (ds *DataSource) CreateReview(ctx context.Context, input model.NewReview) (*model.Review, error) {
	var book model.Book
	result := ds.DB.Where("id = ?", input.BookID).Limit(1).Find(&book)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("book with id '%v' does not exist", input.BookID)
	}
	review := &model.Review{
		BookID: input.BookID,
		Star:   input.Star,
		Text:   input.Text,
	}
	result = ds.DB.Create(review)
	if result.Error != nil {
		return review, result.Error
	} else if result.RowsAffected == 1 {
		return review, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) BookReviews(ctx context.Context, obj *model.Book, offset *int, limit *int, filter *model.ReviewFilter) ([]*model.Review, error) {
	fields := []string{"book_id"}
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		if f.Name != "book_id" {
			fields = append(fields, f.Name)
		}
	}
	var scopeFn = func(bookIDs []int, offset *int, limit *int, filter *model.ReviewFilter) func(tx *gorm.DB) *gorm.DB {
		return func(tx *gorm.DB) *gorm.DB {
			tx.Where("book_id IN ?", bookIDs)
			if filter != nil {
				if filter.Star != nil {
					FilterIntRange(filter.Star, tx, `"reviews"."star"`)
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
	tx := ds.DB.Session(&gorm.Session{DryRun: true}).Select(fields).Scopes(scopeFn([]int{obj.ID}, offset, limit, filter)).Find(&model.Review{})
	group := tx.Statement.SQL.String()
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	var queryFn = func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Book).ID
		}
		var reviews []*model.Review
		result := ds.DB.Select(fields).Scopes(scopeFn(ids, offset, limit, filter)).Find(&reviews)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: reviews,
		}
	}
	filterFn := func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		if groupResults.Error != nil {
			return groupResults
		}
		book := key.Param.(*model.Book)
		greviews := groupResults.Data.([]*model.Review)
		var reviews []*model.Review
		for _, r := range greviews {
			if r.BookID == book.ID {
				reviews = append(reviews, r)
			}
		}
		return &dataloader.Result{Data: reviews}
	}
	data, err := ds.BatchLoad(ctx, &group, key, []int{obj.ID}, obj, queryFn, filterFn)
	if data != nil {
		return data.([]*model.Review), err
	}
	return nil, err
}
