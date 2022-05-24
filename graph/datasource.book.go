package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
	"gorm.io/gorm"
)

func (ds *DataSource) CreateBook(ctx context.Context, input model.NewBook) (*model.Book, error) {
	var authors []*model.Author
	result := ds.DB.Where("name IN (?)", input.AuthorsName).Find(&authors)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(authors) != len(input.AuthorsName) {
		missing := []string{}
		for _, a := range input.AuthorsName {
			nfound := true
			for i, il := 0, len(authors); i < il && nfound; i++ {
				if a == authors[i].Name {
					nfound = false
				}
			}
			if nfound {
				missing = append(missing, a)
			}
		}
		return nil, fmt.Errorf("author with name '%s' does not exist", strings.Join(missing, "', '"))
	}
	book := &model.Book{
		Title:   input.Title,
		Authors: authors,
	}
	result = ds.DB.Omit("Authors.*").Create(book)
	if result.Error != nil {
		emsg := result.Error.Error()
		if strings.Contains(emsg, "duplicate key value violates unique constraint") {
			if strings.Contains(emsg, `"books_title_key"`) {
				return book, fmt.Errorf(`duplicate key books.title "%s"`, book.Title)
			}
		}
		return book, result.Error
	} else if result.RowsAffected == 1 {
		return book, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) UpdateBook(ctx context.Context, input model.UpdateBook) (*model.Book, error) {
	var book model.Book
	result := ds.DB.Where("books.id = ?", input.ID).Limit(1).Find(&book)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("book with id '%v' does not exist", input.ID)
	}
	fields := []string{}
	if input.Title != nil {
		book.Title = *input.Title
		fields = append(fields, "title")
	}
	tx := ds.DB.Begin()
	if input.AuthorsName != nil {
		var authors []*model.Author
		result := ds.DB.Where("name IN (?)", input.AuthorsName).Find(&authors)
		if result.Error != nil {
			return nil, result.Error
		}
		if len(authors) != len(input.AuthorsName) {
			missing := []string{}
			for _, a := range input.AuthorsName {
				nfound := true
				for i, il := 0, len(authors); i < il && nfound; i++ {
					if a == authors[i].Name {
						nfound = false
					}
				}
				if nfound {
					missing = append(missing, a)
				}
			}
			tx.Rollback()
			return nil, fmt.Errorf("author with name '%s' does not exist", strings.Join(missing, "', '"))
		}
		book.Authors = authors
		fields = append(fields, "Authors")
		var authorIDs []int
		for _, a := range book.Authors {
			authorIDs = append(authorIDs, a.ID)
		}
		result = tx.Exec("DELETE FROM book_authors WHERE book_id = ? AND author_id NOT IN (?)", book.ID, authorIDs)
		if result.Error != nil {
			tx.Rollback()
			return &book, result.Error
		}
	}
	result = tx.Select(fields).Omit("Authors.*").Updates(&book)
	if result.Error != nil {
		emsg := result.Error.Error()
		if strings.Contains(emsg, "duplicate key value violates unique constraint") {
			if strings.Contains(emsg, `"books_title_key"`) {
				tx.Rollback()
				return &book, fmt.Errorf(`duplicate key books.title "%s"`, book.Title)
			}
		}
		tx.Rollback()
		return &book, result.Error
	} else if result.RowsAffected == 1 {
		result = tx.Commit()
		return &book, result.Error
	}
	tx.Rollback()
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) DeleteBook(ctx context.Context, id int) (*model.Book, error) {
	var book model.Book
	result := ds.DB.Where("books.id = ?", id).Limit(1).Find(&book)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("book with id '%v' does not exist", id)
	}
	result = ds.DB.Delete(&book)
	if result.Error != nil {
		return &book, result.Error
	} else if result.RowsAffected == 1 {
		return &book, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) Books(ctx context.Context, offset *int, limit *int, filter *model.BookFilter) (*model.BookList, error) {
	needCount := false
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		switch f.Name {
		case "count":
			needCount = true
		case "list":
			for _, f := range graphql.CollectFields(graphql.GetOperationContext(ctx), f.SelectionSet, nil) {
				switch f.Name {
				case "authors", "reviews":
				default:
					fields = append(fields, fmt.Sprintf(`"books"."%s"`, f.Name))
				}
			}
		}
	}
	var scopeFn = func(offset *int, limit *int) func(tx *gorm.DB) *gorm.DB {
		return func(tx *gorm.DB) *gorm.DB {
			tx.Model(&model.Book{})
			if filter != nil {
				if filter.ID != nil {
					tx.Where("books.id = ?", filter.ID)
				}
				if filter.Title != nil {
					FilterText(filter.Title, tx, "books.title")
				}
				if filter.AuthorName != nil {
					sq := ds.DB.Select("book_id")
					sq.Joins("JOIN book_authors ON authors.id = book_authors.author_id")
					op := FilterSubQueryText(filter.AuthorName, sq, `authors.name`)
					tx.Where(fmt.Sprintf("books.id %s (?)", op), sq.Model(&model.Author{}))
				}
				if filter.Star != nil && (filter.Star.Min != nil || filter.Star.Max != nil) {
					tx.Distinct()
					tx.Joins("JOIN reviews ON books.id = reviews.book_id")
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
	tx := ds.DB.Session(&gorm.Session{DryRun: true}).Select(fields).Scopes(scopeFn(offset, limit)).Find(&model.Book{})
	group := tx.Statement.SQL.String()
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	var queryFn = func(keys []*BatchLoaderKey) *dataloader.Result {
		var books []*model.Book
		var count int64
		var result *gorm.DB
		if needCount {
			result = ds.DB.Scopes(scopeFn(nil, nil)).Table("books").Count(&count)
			if result.Error != nil {
				return &dataloader.Result{
					Error: result.Error,
				}
			}
			if count == 0 {
				return &dataloader.Result{
					Data: &model.BookList{List: []*model.Book{}, Count: int(count)},
				}
			}
		}
		result = ds.DB.Select(fields).Scopes(scopeFn(offset, limit)).Find(&books)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: &model.BookList{List: books, Count: int(count)},
		}
	}
	data, err := ds.BatchLoad(ctx, &group, key, nil, nil, queryFn, nil)
	if data != nil {
		return data.(*model.BookList), err
	}
	return nil, err
}

func (ds *DataSource) ReviewBook(ctx context.Context, obj *model.Review) (*model.Book, error) {
	fields := []string{}
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		switch f.Name {
		case "authors", "reviews":
		default:
			fields = append(fields, fmt.Sprintf(`"books"."%s"`, f.Name))
		}
	}
	var scopeFn = func(bookIDs []int) func(tx *gorm.DB) *gorm.DB {
		return func(tx *gorm.DB) *gorm.DB {
			tx.Where("books.id IN ?", bookIDs)
			return tx
		}
	}
	tx := ds.DB.Session(&gorm.Session{DryRun: true}).Select(fields).Scopes(scopeFn([]int{obj.BookID})).Find(&model.Book{})
	group := tx.Statement.SQL.String()
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	var queryFn = func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Review).BookID
		}
		var books []*model.Book
		result := ds.DB.Select(fields).Scopes(scopeFn(ids)).Find(&books)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: books,
		}
	}
	filterFn := func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		if groupResults.Error != nil {
			return groupResults
		}
		review := key.Param.(*model.Review)
		gbooks := groupResults.Data.([]*model.Book)
		for _, b := range gbooks {
			if review.BookID == b.ID {
				return &dataloader.Result{Data: b}
			}
		}
		return &dataloader.Result{Data: nil}
	}
	data, err := ds.BatchLoad(ctx, &group, key, []int{obj.ID}, obj, queryFn, filterFn)
	if data != nil {
		return data.(*model.Book), err
	}
	return nil, err
}
