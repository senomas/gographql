package data

import (
	"context"

	"github.com/graph-gophers/dataloader"
	"github.com/graphql-go/graphql"
)

type Review struct {
	ID     int
	Star   int
	Body   string
	BookID int `json:"book_id"`
}

func (l *Loader) getReviews(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return nil
}

func (l *Loader) LoadReviews(ctx context.Context, p graphql.ResolveParams) (interface{}, error) {
	thunk := l.reviewsLoader.Load(ctx, NewDataKey(p))
	if res, err := thunk(); res != nil {
		if p.Info.FieldName == "reviews" {
			return res.([]*Review), err
		}
		return res.([]*Review)[0], err
	} else {
		return nil, err
	}
}
