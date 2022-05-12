package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/senomas/gographql/graph/generated"
	"github.com/senomas/gographql/graph/model"
)

func (r *bookResolver) Author(ctx context.Context, obj *model.Book) (*model.Author, error) {
	return ctx.Value(Context_DataSource).(*DataSource).BookAuthor(ctx, obj)
}

func (r *bookResolver) Reviews(ctx context.Context, obj *model.Book, offset *int, limit *int, filter *model.ReviewFilter) ([]*model.Review, error) {
	return ctx.Value(Context_DataSource).(*DataSource).BookReviews(ctx, obj, offset, limit, filter)
}

func (r *mutationResolver) CreateAuthor(ctx context.Context, input model.NewAuthor) (*model.Author, error) {
	return ctx.Value(Context_DataSource).(*DataSource).CreateAuthor(ctx, input)
}

func (r *mutationResolver) CreateBook(ctx context.Context, input model.NewBook) (*model.Book, error) {
	return ctx.Value(Context_DataSource).(*DataSource).CreateBook(ctx, input)
}

func (r *mutationResolver) UpdateBook(ctx context.Context, input model.UpdateBook) (*model.Book, error) {
	return ctx.Value(Context_DataSource).(*DataSource).UpdateBook(ctx, input)
}

func (r *mutationResolver) DeleteBook(ctx context.Context, id int) (*model.Book, error) {
	return ctx.Value(Context_DataSource).(*DataSource).DeleteBook(ctx, id)
}

func (r *mutationResolver) CreateReview(ctx context.Context, input model.NewReview) (*model.Review, error) {
	return ctx.Value(Context_DataSource).(*DataSource).CreateReview(ctx, input)
}

func (r *queryResolver) Authors(ctx context.Context, offset *int, limit *int, filter *model.AuthorFilter) ([]*model.Author, error) {
	return ctx.Value(Context_DataSource).(*DataSource).Authors(ctx, offset, limit, filter)
}

func (r *queryResolver) Books(ctx context.Context, offset *int, limit *int, filter *model.BookFilter) ([]*model.Book, error) {
	return ctx.Value(Context_DataSource).(*DataSource).Books(ctx, offset, limit, filter)
}

func (r *reviewResolver) Book(ctx context.Context, obj *model.Review) (*model.Book, error) {
	return ctx.Value(Context_DataSource).(*DataSource).ReviewBook(ctx, obj)
}

// Book returns generated.BookResolver implementation.
func (r *Resolver) Book() generated.BookResolver { return &bookResolver{r} }

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Review returns generated.ReviewResolver implementation.
func (r *Resolver) Review() generated.ReviewResolver { return &reviewResolver{r} }

type bookResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type reviewResolver struct{ *Resolver }
