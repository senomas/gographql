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

func getFields(set ast.SelectionSet, path int, fn func(fields []string, path int, name string) (int, []string), fields []string) []string {
	for _, s := range set {
		switch v := s.(type) {
		case *ast.Field:
			npath, nfields := fn(fields, path, v.Name)
			if nfields != nil {
				fields = nfields
				if v.SelectionSet != nil {
					fields = getFields(v.SelectionSet, npath, fn, fields)
				}
			}
		}
	}
	return fields
}

func GetFields(ctx context.Context, fn func(fields []string, path int, name string) (int, []string)) []string {
	return getFields(graphql.GetFieldContext(ctx).Field.SelectionSet, 0, fn, []string{})
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
	fields := GetFields(ctx, func(fields []string, path int, name string) (int, []string) {
		switch path {
		case 0:
			switch name {
			case "count":
				needCount = true
				return path, fields
			case "list":
				return 1, fields
			}
		case 1: // path: list
			switch name {
			case "authors":
				return path, nil
			case "reviews":
				return path, nil
			default:
				fields = append(fields, fmt.Sprintf(`"books"."%s"`, name))
			}
		}
		return path, fields
	})
	var books []*model.Book
	var count int64
	var queryScope = func(param interface{}) func(db *gorm.DB) *gorm.DB {
		return func(tx *gorm.DB) *gorm.DB {
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
			return tx
		}
	}
	key, err := ds.NewBatchLoaderKey(func(db *gorm.DB, param interface{}) *gorm.DB {
		if needCount {
			db.Scopes(queryScope(param)).Model(&model.Book{}).Count(&count)
		}
		if db.DryRun || !needCount || count > 0 {
			tx := db.Select(fields).Scopes(queryScope(param))
			if offset != nil {
				tx = tx.Offset(*offset)
			}
			if limit != nil {
				tx = tx.Limit(*limit)
			}
			return tx.Find(&books)
		}
		return db
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
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, key)()
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
	type Param struct {
		bookID []int
	}
	type bookAuthor struct {
		Book_ID int
		ID      int
		Name    string
	}

	var authors []*bookAuthor
	queryFn := func(db *gorm.DB, _param interface{}) *gorm.DB {
		param := _param.(*Param)
		return db.Select(fields).Joins("JOIN book_authors ON authors.id = book_authors.author_id").Where("book_authors.book_id IN ?", param.bookID).Model(&model.Author{}).Find(&authors)
	}
	groupFn := func(tx *gorm.DB) *string {
		g := tx.Statement.SQL.String()
		return &g
	}
	findFn := func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Book).ID
		}
		result := keys[0].Query(ds.DB, &Param{
			bookID: ids,
		})
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
	key, err := ds.NewBatchLoaderKey(queryFn, groupFn, &Param{
		bookID: []int{obj.ID},
	}, obj, findFn, filterFn)
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, key)()
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
	type Param struct {
		bookID []int
		offset *int
		limit  *int
		filter *model.ReviewFilter
	}
	var reviews []*model.Review
	queryFn := func(db *gorm.DB, _param interface{}) *gorm.DB {
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
	}
	groupFn := func(tx *gorm.DB) *string {
		g := tx.Statement.SQL.String()
		return &g
	}
	findFn := func(keys []*BatchLoaderKey) *dataloader.Result {
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
	key, err := ds.NewBatchLoaderKey(queryFn, groupFn, &Param{
		bookID: []int{obj.ID},
		offset: offset,
		limit:  limit,
		filter: filter,
	}, obj, findFn, filterFn)
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.([]*model.Review), err
	}
	return nil, err
}

func (ds *DataSource) ReviewBook(ctx context.Context, obj *model.Review) (*model.Book, error) {
	fields := GetFields(ctx, func(fields []string, path int, name string) (int, []string) {
		switch path {
		case 0:
			switch name {
			case "authors":
				return path, nil
			case "reviews":
				return path, nil
			default:
				fields = append(fields, fmt.Sprintf(`"books"."%s"`, name))
			}
		}
		return path, fields
	})
	var books []*model.Book
	key, err := ds.NewBatchLoaderKey(func(db *gorm.DB, param interface{}) *gorm.DB {
		tx := db.Select(fields).Where("books.id IN ?", param)
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
	if err != nil {
		return nil, err
	}
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.(*model.Book), err
	}
	return nil, err
}

func (ds *DataSource) NewBatchLoaderKey(query func(tx *gorm.DB, param interface{}) *gorm.DB, groupFn func(tx *gorm.DB) *string, params interface{}, obj interface{}, find func(keys []*BatchLoaderKey) *dataloader.Result, filter func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result) (*BatchLoaderKey, error) {
	tx := query(ds.DB.Session(&gorm.Session{DryRun: true}), params)
	if tx.Error != nil {
		panic(fmt.Sprintf("invalid query %v", tx.Error))
	}
	key := ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...)
	if key == "" {
		panic(fmt.Sprintf("invalid key %s", key))
	}
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
			result := key.Find(gkeys)
			for _, g := range gkeys {
				results[g.idx] = g.Filter(g, result)
			}
		}
	}
	return results
}
