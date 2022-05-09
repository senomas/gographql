package graph

import (
	"encoding/json"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/senomas/gographql/graph/generated"
	"github.com/stretchr/testify/assert"
)

func TestTodo(t *testing.T) {

	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{}}))
	c := client.New(h)

	t.Run("create todo", func(t *testing.T) {
		var resp struct {
			CreateTodo struct {
				ID   string
				Text string
				Done bool
			}
		}
		c.MustPost(`mutation {
			createTodo(input: {
				text: "Halo"
				userId: "15"
			}) {
				id
				text
				done
			}
		}`, &resp)
		jsonMatch(t, `{
			"CreateTodo": {
				"ID": "1",
				"Text": "Halo",
				"Done": false
			}
		}`, &resp)
	})

	t.Run("find todos", func(t *testing.T) {
		var resp struct {
			Todos []struct {
				ID   string
				Text string
				Done bool
			}
		}
		c.MustPost(`{
			todos {
				id
				text
				done
			}
		}`, &resp)

		jsonMatch(t, `{
			"Todos": [
				{
					"Done": false,
					"ID": "1",
					"Text": "Halo"
				}
			]
		}`, &resp)
	})
}

func jsonMatch(t *testing.T, expected string, resp interface{}) {
	rb, _ := json.MarshalIndent(resp, "", "\t")
	rv := make(map[string]interface{})
	json.Unmarshal([]byte(rb), &rv)
	rJSON, _ := json.MarshalIndent(rv, "", "\t")

	v := make(map[string]interface{})
	json.Unmarshal([]byte(expected), &v)
	eJSON, _ := json.MarshalIndent(v, "", "\t")

	assert.Equal(t, string(eJSON), string(rJSON))
}
