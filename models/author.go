package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/senomas/gographql/data"
)

var AuthorType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Author",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

func AuthorMutations(fields graphql.Fields) graphql.Fields {
	fields["createAuthor"] = &graphql.Field{
		Type:        AuthorType,
		Description: "create new author",
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: CreateAuthorResolver,
	}
	fields["updateAuthor"] = &graphql.Field{
		Type:        AuthorType,
		Description: "update author",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			"name": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: UpdateAuthorResolver,
	}
	return fields
}

func CreateAuthorResolver(p graphql.ResolveParams) (interface{}, error) {
	db := p.Context.Value(ContextKeyDB).(*sql.DB)
	var params []interface{}
	if v, ok := p.Args["name"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter name required")
	}
	if tx, err := db.Begin(); err != nil {
		return nil, fmt.Errorf("failed begin tx, err %v", err)
	} else {
		if rows, err := tx.Query("INSERT INTO authors (name) VALUES ($1) RETURNING id", params...); err != nil {
			return nil, fmt.Errorf("failed to create author, err %v", err)
		} else {
			var id int
			{
				defer rows.Close()
				if rows.Next() {
					rows.Scan(&id)
				} else {
					return nil, errors.New("failed to insert authors")
				}
				tx.Commit()
			}

			if rows, err := db.Query("SELECT id, name FROM authors WHERE id = $1", id); err != nil {
				return nil, err
			} else {
				if rows.Next() {
					var author data.Author
					rows.Scan(&author.ID, &author.Name)

					return author, nil
				}
			}
		}
	}
	return nil, nil
}

func UpdateAuthorResolver(p graphql.ResolveParams) (interface{}, error) {
	db := p.Context.Value(ContextKeyDB).(*sql.DB)
	var params []interface{}
	if v, ok := p.Args["id"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter id required")
	}
	if v, ok := p.Args["name"]; ok {
		params = append(params, v)
	} else {
		return nil, errors.New("parameter name required")
	}
	if tx, err := db.Begin(); err != nil {
		return nil, fmt.Errorf("failed begin tx, err %v", err)
	} else {
		if result, err := tx.Exec("UPDATE authors SET name = $2 WHERE id = $1", params...); err != nil {
			return nil, fmt.Errorf("failed to create author, err %v", err)
		} else {
			tx.Commit()

			if affected, err := result.RowsAffected(); err != nil {
				return nil, err
			} else if affected == 1 {
				if rows, err := db.Query("SELECT id, name FROM authors WHERE id = $1", params[0]); err != nil {
					return nil, err
				} else {
					if rows.Next() {
						var author data.Author
						rows.Scan(&author.ID, &author.Name)

						return author, nil
					}
				}
			} else {
				return nil, fmt.Errorf("affected rows %v", affected)
			}
		}
	}
	return nil, nil
}
