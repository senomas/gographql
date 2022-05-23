package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
	"gorm.io/gorm"
)

func (ds *DataSource) CreateAuthor(ctx context.Context, input model.NewAuthor) (*model.Author, error) {
	author := &model.Author{
		Name: input.Name,
	}
	result := ds.DB.Create(author)
	if result.Error != nil {
		return author, result.Error
	} else if result.RowsAffected == 1 {
		return author, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) Authors(ctx context.Context, offset *int, limit *int, filter *model.AuthorFilter) (*model.AuthorList, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		fields = append(fields, f.Name)
	}
	var authors []*model.Author
	var count int64
	result := ds.DB.Select(fields).Find(&authors).Count(&count)
	if result.Error != nil {
		return nil, result.Error
	}

	return &model.AuthorList{List: authors, Count: int(count)}, nil
}

func (ds *DataSource) BookAuthors(ctx context.Context, obj *model.Book) ([]*model.Author, error) {
	fields := []string{`"book_authors"."book_id"`}
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		fields = append(fields, fmt.Sprintf(`"authors"."%s"`, f.Name))
	}
	type bookAuthor struct {
		Book_ID int
		ID      int
		Name    string
	}
	var scopeFn = func(bookIDs []int) func(tx *gorm.DB) *gorm.DB {
		return func(tx *gorm.DB) *gorm.DB {
			tx.Model(&model.Author{})
			tx.Joins("JOIN book_authors ON authors.id = book_authors.author_id")
			tx.Where("book_authors.book_id IN ?", bookIDs)
			return tx
		}
	}
	tx := ds.DB.Session(&gorm.Session{DryRun: true}).Scopes(scopeFn([]int{obj.ID})).Find(&model.Book{})
	group := tx.Statement.SQL.String()
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	var queryFn = func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Book).ID
		}
		var authors []*bookAuthor
		result := ds.DB.Select(fields).Scopes(scopeFn(ids)).Find(&authors)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: authors,
		}
	}
	filterFn := func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		if groupResults.Error != nil {
			return groupResults
		}
		book := key.Param.(*model.Book)
		gauthors := groupResults.Data.([]*bookAuthor)
		authors := []*model.Author{}
		for _, a := range gauthors {
			if a.Book_ID == book.ID {
				authors = append(authors, &model.Author{
					ID:   a.ID,
					Name: a.Name,
				})
			}
		}
		return &dataloader.Result{Data: authors}
	}
	data, err := ds.BatchLoad(ctx, &group, key, []int{obj.ID}, obj, queryFn, filterFn)
	if data != nil {
		return data.([]*model.Author), err
	}
	return nil, err
}
