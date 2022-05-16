package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/senomas/gographql/graph"
	"github.com/senomas/gographql/graph/generated"
	"gorm.io/gorm"
)

const defaultPort = "8088"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	var db *gorm.DB

	if _, _db, err := graph.Setup(); err != nil {
		log.Panicf("setup database error %v", err)
	} else {
		db = _db
	}

	cfg := generated.Config{Resolvers: &graph.Resolver{}}
	cfg.Directives.Gorm = graph.Directive_Gorm
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))
	var xsrv http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		r.Context()
		srv.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), graph.Context_DataSource, graph.NewDataSource(db))))
	}

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", xsrv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
