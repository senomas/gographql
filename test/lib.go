package test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
)

func QuoteMeta(r string) string {
	return "^" + regexp.QuoteMeta(r) + "$"
}

func QLTest(t *testing.T, schema graphql.Schema, ctx context.Context) (func(query string, str string) *graphql.Result, func(query string, str string) *graphql.Result) {
	return func(query string, str string) *graphql.Result {
			params := graphql.Params{
				Schema:        schema,
				RequestString: query,
				RootObject:    make(map[string]interface{}),
				Context:       ctx,
			}
			r := graphql.Do(params)

			rJSON, _ := json.MarshalIndent(r, "", "\t")

			v := make(map[string]interface{})
			json.Unmarshal([]byte(str), &v)
			eJSON, _ := json.MarshalIndent(v, "", "\t")

			assert.Equal(t, string(eJSON), string(rJSON))

			return r
		}, func(query string, str string) *graphql.Result {
			params := graphql.Params{
				Schema:        schema,
				RequestString: query,
				RootObject:    make(map[string]interface{}),
				Context:       ctx,
			}
			r := graphql.Do(params)

			assert.Equal(t, len(r.Errors), 1)
			assert.ErrorContains(t, r.Errors[0], str)

			return r
		}
}
