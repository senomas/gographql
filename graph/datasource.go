package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/senomas/gographql/graph/model"
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
	idx         int
	key         string
	group       *string
	queryOffset *int
	queryLimit  *int
	Param       interface{}
	processed   bool
	Query       func(db *gorm.DB, params interface{}) *gorm.DB
	Find        func(keys []*BatchLoaderKey) *dataloader.Result
	Filter      func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result
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
	result := ds.DB.Select([]string{"id"}).Where("name = ?", input.AuthorName).Limit(1).Find(&author)
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
	if result.Error != nil {
		return book, result.Error
	} else if result.RowsAffected == 1 {
		return book, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) CreateReview(ctx context.Context, input model.NewReview) (*model.Review, error) {
	review := &model.Review{
		BookID: input.BookID,
		Star:   input.Star,
		Text:   input.Text,
	}
	result := ds.DB.Create(review)
	if result.Error != nil {
		return review, result.Error
	} else if result.RowsAffected == 1 {
		return review, nil
	}
	return nil, fmt.Errorf("RowsAffected %v", result.RowsAffected)
}

func (ds *DataSource) Authors(ctx context.Context, queryOffset *int, queryLimit *int, id *int, name *string) ([]*model.Author, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		fields = append(fields, f.Name)
	}
	var authors []*model.Author
	result := ds.DB.Select(fields).Find(&authors)
	if result.Error != nil {
		return nil, result.Error
	}

	return authors, nil
}

func (ds *DataSource) Books(ctx context.Context, queryOffset *int, queryLimit *int, id *int, title *string, authorName *string, minStar *int, maxStar *int) ([]*model.Book, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		if f.Name == "author" {
			fields = append(fields, "books.author_id")
		} else if f.Name == "reviews" {
			// no field
		} else {
			fields = append(fields, "books."+f.Name)
		}
	}
	var books []*model.Book
	key := ds.NewBatchLoaderKey(func(db *gorm.DB, param interface{}) *gorm.DB {
		tx := db.Select(fields)
		if queryOffset != nil {
			tx = tx.Offset(*queryOffset)
		}
		if queryLimit != nil {
			tx = tx.Limit(*queryLimit)
		}
		if id != nil {
			tx.Where("books.id = ?", id)
		}
		if title != nil {
			tx.Where("books.title LIKE ?", title)
		}
		if authorName != nil {
			tx.Where("author_id = (?)", ds.DB.Select("id").Where("name LIKE ?", authorName).Table("authors"))
		}
		if minStar != nil || maxStar != nil {
			tx.Distinct()
			tx.Joins("JOIN reviews ON books.id = reviews.book_id")
			if minStar != nil {
				tx.Where("reviews.star >= ?", minStar)
			}
			if maxStar != nil {
				tx.Where("reviews.star <= ?", maxStar)
			}
		}
		return tx.Find(&books)
	}, nil, nil, nil, nil, nil, func(keys []*BatchLoaderKey) *dataloader.Result {
		result := keys[0].Query(ds.DB, nil)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: books,
		}
	}, func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		return groupResults
	})
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.([]*model.Book), err
	}
	return nil, err
}

func (ds *DataSource) BookAuthor(ctx context.Context, obj *model.Book) (*model.Author, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		fields = append(fields, f.Name)
	}
	var authors []*model.Author
	key := ds.NewBatchLoaderKey(func(db *gorm.DB, param interface{}) *gorm.DB {
		return db.Select(fields).Where("id IN ?", param).Find(&authors)
	}, func(tx *gorm.DB) *string {
		g := tx.Statement.SQL.String()
		return &g
	}, []interface{}{obj.AuthorID}, obj, nil, nil, func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Book).AuthorID
		}
		result := keys[0].Query(ds.DB, ids)
		if result.Error != nil {
			return &dataloader.Result{
				Error: result.Error,
			}
		}
		return &dataloader.Result{
			Data: authors,
		}
	}, func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
		if groupResults.Error != nil {
			return groupResults
		}
		book := key.Param.(*model.Book)
		authors := groupResults.Data.([]*model.Author)
		for _, a := range authors {
			if a.ID == book.AuthorID {
				return &dataloader.Result{Data: a}
			}
		}
		return &dataloader.Result{}
	})
	data, err := ds.BatchLoader.Load(ctx, key)()
	if data != nil {
		return data.(*model.Author), err
	}
	return nil, err
}

func (ds *DataSource) BookReviews(ctx context.Context, obj *model.Book, queryOffset *int, queryLimit *int, minStar *int, maxStar *int) ([]*model.Review, error) {
	fields := []string{"book_id"}
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		if f.Name != "book_id" {
			fields = append(fields, f.Name)
		}
	}
	type Param struct {
		bookID      []int
		queryOffset *int
		queryLimit  *int
		minStar     *int
		maxStar     *int
	}
	var reviews []*model.Review
	key := ds.NewBatchLoaderKey(func(db *gorm.DB, _param interface{}) *gorm.DB {
		param := _param.(*Param)
		tx := db.Select(fields).Where("book_id IN ?", param.bookID)
		if queryOffset != nil {
			tx = tx.Offset(*param.queryOffset)
		}
		if queryLimit != nil {
			tx = tx.Limit(*param.queryLimit)
		}
		if minStar != nil {
			tx = tx.Where("star >= ?", param.minStar)
		}
		if maxStar != nil {
			tx = tx.Where("star <= ?", param.maxStar)
		}
		return tx.Find(&reviews)
	}, func(tx *gorm.DB) *string {
		g := tx.Statement.SQL.String()
		return &g
	}, &Param{
		bookID:      []int{obj.ID},
		queryOffset: queryOffset,
		queryLimit:  queryLimit,
		minStar:     minStar,
		maxStar:     maxStar,
	}, obj, queryOffset, queryLimit, func(keys []*BatchLoaderKey) *dataloader.Result {
		ids := make([]int, len(keys))
		for i, k := range keys {
			ids[i] = k.Param.(*model.Book).ID
		}
		result := keys[0].Query(ds.DB, &Param{
			bookID:      ids,
			queryOffset: queryOffset,
			queryLimit:  queryLimit,
			minStar:     minStar,
			maxStar:     maxStar,
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

func (ds *DataSource) NewBatchLoaderKey(query func(tx *gorm.DB, param interface{}) *gorm.DB, groupFn func(tx *gorm.DB) *string, params interface{}, obj interface{}, queryOffset *int, queryLimit *int, find func(keys []*BatchLoaderKey) *dataloader.Result, filter func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result) *BatchLoaderKey {
	tx := query(ds.DB.Session(&gorm.Session{DryRun: true}), params)
	var group *string
	if groupFn != nil {
		group = groupFn(tx)
	}
	return &BatchLoaderKey{
		key:         ds.DB.Dialector.Explain(tx.Statement.SQL.String(), tx.Statement.Vars...),
		group:       group,
		queryOffset: queryOffset,
		queryLimit:  queryLimit,
		Query:       query,
		Param:       obj,
		Find:        find,
		Filter:      filter,
	}
}

func (ds *DataSource) batchLoader(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	// for i, k := range keys {
	// 	key := k.(*BatchLoaderKey)
	// 	fmt.Printf("BATCH LOADER %v : [%v]  [%v]\n", i, key.group, key.key)
	// }
	results := make([]*dataloader.Result, len(keys))
	keysLen := len(keys)
	for i := 0; i < keysLen; i++ {
		key := keys[i].(*BatchLoaderKey)
		if !key.processed {
			key.idx = i
			gkeys := []*BatchLoaderKey{key}
			if key.group != nil && key.queryLimit == nil && key.queryOffset == nil {
				for j := i + 1; j < keysLen; j++ {
					jkey := keys[j].(*BatchLoaderKey)
					if !jkey.processed {
						if *key.group == *jkey.group && jkey.queryLimit == nil && jkey.queryOffset == nil {
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
