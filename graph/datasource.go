package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
	"gorm.io/gorm"
)

type ContextID string

const Context_DataSource = ContextID("DataSource")

type DataSource struct {
	DB          *gorm.DB
	BatchLoader *dataloader.Loader
}

type BatchLoaderKey struct {
	idx           int
	key           string
	group         *string
	processed     bool
	queryFn       func(keys []*BatchLoaderKey) *dataloader.Result
	queryFilterFn func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result
	Param         interface{}
}

func (d *BatchLoaderKey) String() string {
	return d.key
}

func (d *BatchLoaderKey) Raw() interface{} {
	return d
}

func NewDataSource(db *gorm.DB) *DataSource {
	d := DataSource{DB: db}
	d.BatchLoader = dataloader.NewBatchedLoader(d.batchLoader, dataloader.WithWait(100*time.Millisecond))
	return &d
}

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
					tx.Distinct()
					tx.Joins("JOIN book_authors ON books.id = book_authors.book_id LEFT JOIN authors ON authors.id = book_authors.author_id")
					FilterText(filter.AuthorName, tx, `"authors".name`)
				}
				if filter.Star != nil && (filter.Star.Min != nil || filter.Star.Max != nil) {
					tx.Distinct()
					tx.Joins("JOIN reviews ON books.id = reviews.book_id")
					if filter.Star.Min != nil {
						tx.Where("reviews.star >= ?", filter.Star.Min)
					}
					if filter.Star.Max != nil {
						tx.Where("reviews.star <= ?", filter.Star.Max)
					}
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
	loaderKey, err := ds.NewBatchLoaderKey(&group, key, nil, nil, queryFn, nil)
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, loaderKey)()
	if data != nil {
		return data.(*model.BookList), err
	}
	return nil, err
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
	loaderKey, err := ds.NewBatchLoaderKey(&group, key, []int{obj.ID}, obj, queryFn, filterFn)
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, loaderKey)()
	if data != nil {
		return data.([]*model.Author), err
	}
	return nil, err
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
					if filter.Star.Min != nil {
						tx.Where("star >= ?", filter.Star.Min)
					}
					if filter.Star.Max != nil {
						tx.Where("star <= ?", filter.Star.Max)
					}
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
	loaderKey, err := ds.NewBatchLoaderKey(&group, key, []int{obj.ID}, obj, queryFn, filterFn)
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, loaderKey)()
	if data != nil {
		return data.([]*model.Review), err
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
	loaderKey, err := ds.NewBatchLoaderKey(&group, key, []int{obj.ID}, obj, queryFn, filterFn)
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, loaderKey)()
	if data != nil {
		return data.(*model.Book), err
	}
	return nil, err
}

func (ds *DataSource) NewBatchLoaderKey(group *string, key string, params interface{}, obj interface{}, queryFn func(keys []*BatchLoaderKey) *dataloader.Result, queryFilterFn func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result) (*BatchLoaderKey, error) {
	if key == "" {
		panic(fmt.Sprintf("invalid key %s", key))
	}
	return &BatchLoaderKey{
		key:           key,
		group:         group,
		Param:         obj,
		queryFn:       queryFn,
		queryFilterFn: queryFilterFn,
	}, nil
}

func (ds *DataSource) batchLoader(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	keysLen := len(keys)
	for i := 0; i < keysLen; i++ {
		key := keys[i].(*BatchLoaderKey)
		if !key.processed {
			key.idx = i
			gkeys := []*BatchLoaderKey{key}
			if key.group != nil {
				for j := i + 1; j < keysLen; j++ {
					jkey := keys[j].(*BatchLoaderKey)
					if !jkey.processed {
						if *key.group == *jkey.group {
							jkey.idx = j
							jkey.processed = true
							gkeys = append(gkeys, jkey)
						}
					}
				}
			}
			result := key.queryFn(gkeys)
			for _, g := range gkeys {
				if g.queryFilterFn != nil {
					results[g.idx] = g.queryFilterFn(g, result)
				} else {
					results[g.idx] = result
				}
			}
		}
	}
	return results
}
