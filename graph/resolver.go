package graph

import "github.com/senomas/gographql/graph/model"

//go:generate go run github.com/99designs/gqlgen generate

type Resolver struct {
	todos []*model.Todo
}
