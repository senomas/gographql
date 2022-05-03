package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/graphql-go/graphql"
)

func main() {
	fields := graphql.Fields{
		"hello": &graphql.Field{
			Type: graphql.String,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return "World", nil
			},
		},
	}

	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	var schema graphql.Schema
	if s, err := graphql.NewSchema(schemaConfig); err != nil {
		log.Fatalf("Failed to create new GraphQL Schema, err %v", err)
	} else {
		schema = s
	}

	query := `
	{
		book(id: 2) {
			id
			title
			author {
				name
			}
			reviews {
				star
				body
			}
		}
	}`
	params := graphql.Params{Schema: schema, RequestString: query}

	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		log.Fatalf("Failed to execute graphql operation, errors: %+v", r.Errors)
	}

	rJSON, _ := json.MarshalIndent(r, "", "  ")
	fmt.Printf("%s\n", rJSON)
}
