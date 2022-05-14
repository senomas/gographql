package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
	"github.com/vektah/gqlparser/v2/ast"
	"gorm.io/gorm"
)

type ContextID string

const Context_DataSource = ContextID("DataSource")
const Context_Parent = ContextID("Parent")

type DataSource struct {
	DB          *gorm.DB
	BatchLoader *dataloader.Loader
}

type BatchLoaderKey struct {
	idx       int
	key       string
	group     *string
	processed bool
	Param     interface{}
	Query     func(db *gorm.DB, params interface{}) *gorm.DB
	Find      func(keys []*BatchLoaderKey) *dataloader.Result
	Filter    func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result
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
	var author model.Author
	result := ds.DB.Where("name = ?", input.AuthorName).Limit(1).Find(&author)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("author with name '%v' not exist", input.AuthorName)
	}
	book := &model.Book{
		Title:    input.Title,
		AuthorID: author.ID,
	}
	result = ds.DB.Create(book)
	book.Author = &author
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
	result := ds.DB.Where("books.id = ?", input.ID).Limit(1).Joins("Author").Find(&book)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("book with id '%v' not exist", input.ID)
	}
	fields := []string{}
	if input.Title != nil {
		book.Title = *input.Title
		fields = append(fields, "title")
	}
	if input.AuthorName != nil {
		var author model.Author
		result := ds.DB.Where("name = ?", input.AuthorName).Limit(1).Find(&author)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, fmt.Errorf("author with name '%v' not exist", input.AuthorName)
		}
		book.AuthorID = author.ID
		book.Author = &author
		fields = append(fields, "author_id")
	}
	result = ds.DB.Select(fields).Updates(&book)
	if result.Error != nil {
		emsg := result.Error.Error()
		if strings.Contains(emsg, "duplicate key value violates unique constraint") {
			if strings.Contains(emsg, `"books_title_key"`) {
				return &book, fmt.Errorf(`duplicate key books.title "%s"`, book.Title)
			}
		}
		return &book, result.Error
	} else if result.RowsAffected == 1 {
		return &book, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) DeleteBook(ctx context.Context, id int) (*model.Book, error) {
	var book model.Book
	result := ds.DB.Where("books.id = ?", id).Limit(1).Joins("Author").Find(&book)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("book with id '%v' not exist", id)
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
		return nil, fmt.Errorf("book with id '%v' not exist", input.BookID)
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

type Field struct {
	Name  string
	Path  string
	Field *ast.Field
}

func getFields(set ast.SelectionSet, path string, fields []*Field) []*Field {
	for _, s := range set {
		switch v := s.(type) {
		case *ast.Field:
			npath := path + v.Name
			fields = append(fields, &Field{
				Name:  v.Name,
				Path:  npath,
				Field: v,
			})
		}
	}
	return fields
}

func getRootField(set ast.SelectionSet, root string, path string, fields []*Field) []*Field {
	for _, s := range set {
		switch v := s.(type) {
		case *ast.Field:
			npath := path + v.Name
			if npath == root {
				return getFields(v.SelectionSet, "", fields)
			}
			if v.SelectionSet != nil {
				fields = getRootField(v.SelectionSet, root, npath+".", fields)
			}
		}
	}
	return fields
}

func GetFields(ctx context.Context, root string) []*Field {
	if root == "" {
		return getFields(graphql.GetFieldContext(ctx).Field.SelectionSet, "", []*Field{})
	} else {
		return getRootField(graphql.GetFieldContext(ctx).Field.SelectionSet, root, "", []*Field{})
	}
}

func (ds *DataSource) Books(ctx context.Context, offset *int, limit *int, filter *model.BookFilter) (*model.BookList, error) {
	var fields []string
	needCount := false
	needAuthor := false
	for _, f := range GetFields(ctx, "") {
		if f.Name == "count" {
			needCount = true
		}
	}
	for _, f := range GetFields(ctx, "list") {
		if f.Name == "author" {
			needAuthor = true
			fields = append(fields, "books.author_id")
		} else if f.Name == "reviews" {
			// no field
		} else {
			fields = append(fields, "books."+f.Name)
		}
	}
	var books []*model.Book
	var count int64
	var queryScope = func(param interface{}, needAuthor bool) func(db *gorm.DB) *gorm.DB {
		return func(db *gorm.DB) *gorm.DB {
			tx := db
			if filter != nil {
				if filter.ID != nil {
					tx.Where("books.id = ?", filter.ID)
				}
				if filter.Title != nil {
					FilterText(filter.Title, tx, "books.title")
				}
				if filter.AuthorName != nil {
					needAuthor = true
					FilterText(filter.AuthorName, tx, `"Author".name`)
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
			if needAuthor {
				tx.Joins("Author")
			}
			return tx
		}
	}
	key := ds.NewBatchLoaderKey(func(db *gorm.DB, param interface{}) *gorm.DB {
		if needCount {
			db.Scopes(queryScope(param, false)).Model(&model.Book{}).Count(&count)
		}
		tx := db.Select(fields).Scopes(queryScope(param, needAuthor))
		if offset != nil {
			tx = tx.Offset(*offset)
		}
		if limit != nil {
			tx = tx.Limit(*limit)
		}
		return tx.Find(&books)
	}, nil, nil, nil, func(keys []*BatchLoaderKey) *dataloader.Result {
		result := keys[0].Query(ds.DB, nil)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: &model.BookList{List: books, Count: int(count)},
		}
	}, func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		return groupResults
	})
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.(*model.BookList), err
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
	type Param struct {
		bookID []int
		offset *int
		limit  *int
		filter *model.ReviewFilter
	}
	var reviews []*model.Review
	key := ds.NewBatchLoaderKey(func(db *gorm.DB, _param interface{}) *gorm.DB {
		param := _param.(*Param)
		tx := db.Select(fields).Where("book_id IN ?", param.bookID)
		if param.offset != nil {
			tx = tx.Offset(*param.offset)
		}
		if param.limit != nil {
			tx = tx.Limit(*param.limit)
		}
		if param.filter != nil {
			if param.filter.Star != nil {
				if param.filter.Star.Min != nil {
					tx = tx.Where("star >= ?", param.filter.Star.Min)
				}
				if param.filter.Star.Max != nil {
					tx = tx.Where("star <= ?", param.filter.Star.Max)
				}
			}
		}
		return tx.Find(&reviews)
	}, func(tx *gorm.DB) *string {
		g := tx.Statement.SQL.String()
		return &g
	}, &Param{
		bookID: []int{obj.ID},
		offset: offset,
		limit:  limit,
		filter: filter,
	}, obj, func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Book).ID
		}
		result := keys[0].Query(ds.DB, &Param{
			bookID: ids,
			offset: offset,
			limit:  limit,
			filter: filter,
		})
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: reviews,
		}
	}, func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
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
	})
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.([]*model.Review), err
	}
	return nil, err
}

func (ds *DataSource) ReviewBook(ctx context.Context, obj *model.Review) (*model.Book, error) {
	var fields []string
	needAuthor := false
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		if f.Name == "author" {
			fields = append(fields, "books.author_id")
			needAuthor = true
		} else if f.Name == "reviews" {
			// no field
		} else {
			fields = append(fields, "books."+f.Name)
		}
	}
	var books []*model.Book
	key := ds.NewBatchLoaderKey(func(db *gorm.DB, param interface{}) *gorm.DB {
		tx := db.Select(fields).Where("books.id IN ?", param)
		if needAuthor {
			tx.Joins("Author")
		}
		return tx.Find(&books)
	}, func(tx *gorm.DB) *string {
		g := tx.Statement.SQL.String()
		return &g
	}, []interface{}{obj.BookID}, obj, func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Review).BookID
		}
		result := keys[0].Query(ds.DB, ids)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: books,
		}
	}, func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		if groupResults.Error != nil {
			return groupResults
		}
		review := key.Param.(*model.Review)
		books := groupResults.Data.([]*model.Book)
		for _, b := range books {
			if b.ID == review.BookID {
				return &dataloader.Result{Data: b}
			}
		}
		return &dataloader.Result{}
	})
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.(*model.Book), err
	}
	return nil, err
}

func (ds *DataSource) NewBatchLoaderKey(query func(tx *gorm.DB, param interface{}) *gorm.DB, groupFn func(tx *gorm.DB) *string, params interface{}, obj interface{}, find func(keys []*BatchLoaderKey) *dataloader.Result, filter func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result) *BatchLoaderKey {
	tx := query(ds.DB.Session(&gorm.Session{DryRun: true}), params)
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	var group *string
	if groupFn != nil {
		group = groupFn(tx)
	}
	return &BatchLoaderKey{
		key:    key,
		group:  group,
		Query:  query,
		Param:  obj,
		Find:   find,
		Filter: filter,
	}
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
			result := key.Find(gkeys)
			for _, g := range gkeys {
				results[g.idx] = g.Filter(g, result)
			}
		}
	}
	return results
}
