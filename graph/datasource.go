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
	DB           *gorm.DB
	AuthorLoader *dataloader.Loader
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

func (ds *DataSource) Authors(ctx context.Context) ([]*model.Author, error) {
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

func (ds *DataSource) Books(ctx context.Context) ([]*model.Book, error) {
	var fields []string
	for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
		if f.Name == "author" {
			fields = append(fields, "author_id")
		} else {
			fields = append(fields, f.Name)
		}
	}
	var books []*model.Book
	result := ds.DB.Select(fields).Find(&books)
	if result.Error != nil {
		return nil, result.Error
	}

	return books, nil
}

func (ds *DataSource) BookAuthor(ctx context.Context, obj *model.Book) (*model.Author, error) {
	data, err := ds.AuthorLoader.Load(ctx, &DataLoaderKey{Kind: "BookAuthor.byID", key: fmt.Sprintf("BookAuthor.%v", obj.AuthorID), Parent: obj})()
	return data.(*model.Author), err
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
	// result := ds.DB.Select(fields).Where("id = ?", obj.AuthorID).Find(&author)
	return results
}
