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
	DB            *gorm.DB
	AuthorLoader  *dataloader.Loader
	ReviewsLoader *dataloader.Loader
}

type DataLoaderKey struct {
	key    string
	Kind   string
	Parent interface{}
}

func (d *DataLoaderKey) String() string {
	return d.key
}

func (d *DataLoaderKey) Raw() interface{} {
	return d
}

func NewDataSource(db *gorm.DB) *DataSource {
	d := DataSource{DB: db}
	d.AuthorLoader = dataloader.NewBatchedLoader(d.authorLoader)
	d.ReviewsLoader = dataloader.NewBatchedLoader(d.reviewsLoader)
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
	data, err := ds.AuthorLoader.Load(ctx, &DataLoaderKey{Kind: "BookAuthor.byID", key: fmt.Sprintf("BookAuthor.%v", obj.AuthorID), Parent: obj})()
	return data.(*model.Author), err
}

func (ds *DataSource) BookReviews(ctx context.Context, obj *model.Book) ([]*model.Review, error) {
	data, err := ds.ReviewsLoader.Load(ctx, &DataLoaderKey{Kind: "BookReviews.byID", key: fmt.Sprintf("BookReviews.%v", obj.ID), Parent: obj})()
	return data.([]*model.Review), err
}

func (ds *DataSource) authorLoader(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var byIDs []int
	var byIDixs []int
	for ix, _key := range keys {
		key := _key.(*DataLoaderKey)
		switch key.Kind {
		case "BookAuthor.byID":
			byIDixs = append(byIDixs, ix)
			byIDs = append(byIDs, key.Parent.(*model.Book).AuthorID)
		default:
			fmt.Printf("authorLoader.Kind '%s' not supported!", key.Kind)
		}
	}
	if len(byIDs) > 0 {
		var authors []*model.Author
		result := ds.DB.Where("id = ?", pq.Array(byIDs)).Find(&authors)
		if result.Error != nil {
			for _, ix := range byIDixs {
				results[ix] = &dataloader.Result{
					Error: result.Error,
				}
			}
		} else {
			for i, ix := range byIDixs {
				id := byIDs[i]
				var author *model.Author
				for _, a := range authors {
					if a.ID == id {
						author = a
					}
				}
				results[ix] = &dataloader.Result{
					Data:  author,
					Error: result.Error,
				}
			}
		}
	}
	return results
}

func (ds *DataSource) reviewsLoader(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var byIDs []int
	var byIDixs []int
	for ix, _key := range keys {
		key := _key.(*DataLoaderKey)
		switch key.Kind {
		case "BookReviews.byID":
			byIDixs = append(byIDixs, ix)
			byIDs = append(byIDs, key.Parent.(*model.Book).ID)
		default:
			fmt.Printf("authorLoader.Kind '%s' not supported!", key.Kind)
		}
	}
	if len(byIDs) > 0 {
		var reviews []*model.Review
		result := ds.DB.Where("id = ?", pq.Array(byIDs)).Find(&reviews)
		if result.Error != nil {
			for _, ix := range byIDixs {
				results[ix] = &dataloader.Result{
					Error: result.Error,
				}
			}
		} else {
			for i, ix := range byIDixs {
				id := byIDs[i]
				var breviews []*model.Review
				for _, r := range reviews {
					if r.BookID == id {
						breviews = append(breviews, r)
					}
				}
				results[ix] = &dataloader.Result{
					Data:  breviews,
					Error: result.Error,
				}
			}
		}
	}
	return results
}
