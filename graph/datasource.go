package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/graph-gophers/dataloader"
	"github.com/lib/pq"
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
	group       string
	queryOffset *int
	queryLimit  *int
	Param       interface{}
	processed   bool
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
	d.BatchLoader = dataloader.NewBatchedLoader(d.batchLoader)
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

func (ds *DataSource) Books(ctx context.Context, queryOffset *int, queryLimit *int, id *int, title *string, authorName *string) ([]*model.Book, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		if f.Name == "author" {
			fields = append(fields, "author_id")
		} else if f.Name == "reviews" {
			// no field
		} else {
			fields = append(fields, f.Name)
		}
	}
	var books []*model.Book
	tx := ds.DB.Select(fields)
	if queryOffset != nil {
		tx = tx.Offset(*queryOffset)
	}
	if queryLimit != nil {
		tx = tx.Limit(*queryLimit)
	}
	if id != nil {
		tx.Where("id = ?", id)
	}
	if title != nil {
		tx.Where("title LIKE ?", title)
	}
	if authorName != nil {
		tx.Where("author_id = ?", ds.DB.Select("id").Where("name LIKE ?", authorName).Table("author"))
	}
	result := tx.Find(&books)
	if result.Error != nil {
		return nil, result.Error
	}

	return books, nil
}

func (ds *DataSource) BookAuthor(ctx context.Context, obj *model.Book) (*model.Author, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		fields = append(fields, f.Name)
	}
	var authors []*model.Author
	stmt := ds.DB.Session(&gorm.Session{DryRun: true}).Select(fields).Where("id = ?", obj.AuthorID).Find(&authors).Statement
	group := stmt.SQL.String()
	key := &BatchLoaderKey{
		key:   ds.DB.Dialector.Explain(group, stmt.Vars...),
		group: group,
		Param: obj,
		Find: func(keys []*BatchLoaderKey) *dataloader.Result {
			ids := make([]int, len(keys))
			for i, k := range keys {
				ids[i] = k.Param.(*model.Book).AuthorID
			}
			var authors []*model.Author
			result := ds.DB.Select(fields).Where("id = ?", pq.Array(ids)).Find(&authors)
			if result.Error != nil {
				return &dataloader.Result{
					Error: result.Error,
				}
			}
			return &dataloader.Result{
				Data: authors,
			}
		},
		Filter: func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
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
		},
	}
	data, err := ds.BatchLoader.Load(ctx, key)()
	return data.(*model.Author), err
}

func (ds *DataSource) BookReviews(ctx context.Context, obj *model.Book, queryOffset *int, queryLimit *int, minStar *int, maxStar *int) ([]*model.Review, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		fields = append(fields, f.Name)
	}
	var reviews []*model.Review
	tx := ds.DB.Session(&gorm.Session{DryRun: true}).Select(fields).Where("book_id = ?", obj.ID)
	if queryOffset != nil {
		tx = tx.Offset(*queryOffset)
	}
	if queryLimit != nil {
		tx = tx.Limit(*queryLimit)
	}
	if minStar != nil {
		tx = tx.Where("star >= ?", minStar)
	}
	if maxStar != nil {
		tx = tx.Where("star <= ?", maxStar)
	}
	stmt := tx.Find(&reviews).Statement
	group := stmt.SQL.String()
	key := &BatchLoaderKey{
		key:         ds.DB.Dialector.Explain(group, stmt.Vars...),
		group:       group,
		queryOffset: queryOffset,
		queryLimit:  queryLimit,
		Param:       obj,
		Find: func(keys []*BatchLoaderKey) *dataloader.Result {
			ids := make([]int, len(keys))
			for i, k := range keys {
				ids[i] = k.Param.(*model.Book).ID
			}
			var reviews []*model.Review
			tx := ds.DB.Select(fields).Where("book_id = ?", pq.Array(ids))
			if queryOffset != nil {
				tx = tx.Offset(*queryOffset)
			}
			if queryLimit != nil {
				tx = tx.Limit(*queryLimit)
			}
			if minStar != nil {
				tx = tx.Where("star >= ?", minStar)
			}
			if maxStar != nil {
				tx = tx.Where("star <= ?", maxStar)
			}
			result := tx.Find(&reviews)
			if result.Error != nil {
				return &dataloader.Result{
					Error: result.Error,
				}
			}
			return &dataloader.Result{
				Data: reviews,
			}
		},
		Filter: func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result {
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
		},
	}
	data, err := ds.BatchLoader.Load(ctx, key)()
	return data.([]*model.Review), err
}

func (ds *DataSource) batchLoader(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	for i, k := range keys {
		fmt.Printf("BATCH-LOADER %v : [%s] [%s]", i, k.(*BatchLoaderKey).key, k.(*BatchLoaderKey).group)
	}
	results := make([]*dataloader.Result, len(keys))
	keysLen := len(keys)
	for i := 0; i < keysLen; i++ {
		key := keys[i].(*BatchLoaderKey)
		if !key.processed {
			key.idx = i
			gkeys := []*BatchLoaderKey{key}
			if key.queryLimit == nil && key.queryOffset == nil {
				for j := i + 1; j < keysLen; j++ {
					jkey := keys[j].(*BatchLoaderKey)
					if !jkey.processed {
						if key.group == jkey.group && jkey.queryLimit == nil && jkey.queryOffset == nil {
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
