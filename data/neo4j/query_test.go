package main

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc"

	"github.com/dgraph-io/dgraph/client"
	"github.com/dgraph-io/dgraph/query/graph"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
)

func BenchmarkDgraph(b *testing.B) {
	queries := []struct {
		name  string
		query string
	}{
		{
			"Simple", `{
					me(id: m.06pj8) {
						type.object.name.en
						film.director.film  {
							film.film.genre {
								type.object.name.en
							}
							type.object.name.en
							film.film.initial_release_date
						}
					}
				}`,
		},
		{
			"GetStarted1", `{
				director(allof("type.object.name.en", "steven spielberg")) {
					type.object.name.en
					film.director.film (order: film.film.initial_release_date) {
						type.object.name.en
						film.film.initial_release_date
					}
				}
			}`,
		},
		{
			"GetStarted2", `{
				director(allof("type.object.name.en", "steven spielberg")) {
						type.object.name.en
						film.director.film (order: film.film.initial_release_date) @filter(geq("film.film.initial_release_date", "1984-08")) {
							type.object.name.en
							film.film.initial_release_date
						}
					}
				}`,
		}, {
			"GetStarted3", `{
					director(allof("type.object.name.en", "steven spielberg")) {
						type.object.name.en
						film.director.film (order: film.film.initial_release_date) @filter(geq("film.film.initial_release_date", "1990") && leq("film.film.initial_release_date", "2000")) {
							type.object.name.en
							film.film.initial_release_date
						}
					}
				}`,
		},
	}

	nq := graph.NQuad{
		Subject:     "m.0h_b6x1",
		Predicate:   "type.object.name.en",
		ObjectValue: &graph.Value{&graph.Value_StrVal{"Terminal"}},
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-Query", q.name), func(b *testing.B) {

			conn, err := grpc.Dial("127.0.0.1:8080", grpc.WithInsecure())
			if err != nil {
				b.Fatal("DialTCPConnection")
			}
			c := graph.NewDgraphClient(conn)
			req := client.Req{}
			req.SetQuery(q.query)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := c.Run(context.Background(), req.Request())
				if err != nil {
					b.Fatalf("Error in getting response from server, %s", err)
				}
			}
			conn.Close()
		})
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-Query-parallel", q.name), func(b *testing.B) {

			b.RunParallel(func(pb *testing.PB) {
				conn, err := grpc.Dial("127.0.0.1:8080", grpc.WithInsecure())
				if err != nil {
					b.Fatal("DialTCPConnection")
				}
				c := graph.NewDgraphClient(conn)
				req := client.Req{}
				req.SetQuery(q.query)
				for pb.Next() {
					_, err = c.Run(context.Background(), req.Request())
					if err != nil {
						b.Fatal("Error in query", err)
					}
				}
			})
		})
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-QueryAndMutation", q.name), func(b *testing.B) {

			conn, err := grpc.Dial("127.0.0.1:8080", grpc.WithInsecure())
			if err != nil {
				b.Fatal("DialTCPConnection")
			}
			c := graph.NewDgraphClient(conn)
			req := client.Req{}
			req.SetQuery(q.query)
			req.AddMutation(nq, client.SET)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := c.Run(context.Background(), req.Request())
				if err != nil {
					b.Fatalf("Error in getting response from server, %s", err)
				}
			}
			conn.Close()
		})
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-QueryAndMutation-parallel", q.name), func(b *testing.B) {

			b.RunParallel(func(pb *testing.PB) {
				conn, err := grpc.Dial("127.0.0.1:8080", grpc.WithInsecure())
				if err != nil {
					b.Fatal("DialTCPConnection")
				}
				c := graph.NewDgraphClient(conn)
				req := client.Req{}
				req.SetQuery(q.query)
				req.AddMutation(nq, client.SET)
				for pb.Next() {
					_, err = c.Run(context.Background(), req.Request())
					if err != nil {
						b.Fatal("Error in query", err)
					}
				}
			})
		})
	}
}

func BenchmarkNeo(b *testing.B) {
	queries := []struct {
		name  string
		query string
	}{
		{"Simple", `MATCH (d: Director) - [r:FILMS] -> (f:Film) - [r2:GENRE] -> (g:Genre) WHERE d.directorId="m.06pj8" RETURN d, f, g`},
		{"GetStarted1", `MATCH (d: Director) - [r:FILMS] -> (f:Film) WHERE d.name CONTAINS "Steven Spielberg" WITH d,f ORDER BY f.release_date ASC RETURN d, f`},
		{"GetStarted2", `MATCH (d: Director) - [r:FILMS] -> (f:Film) WHERE d.name CONTAINS "Steven Spielberg" AND f.release_date >= "1984-08" WITH d,f ORDER BY f.release_date ASC RETURN d, f`},
		{"GetStarted3", `MATCH (d: Director) - [r:FILMS] -> (f:Film) WHERE d.name CONTAINS "Steven Spielberg" AND f.release_date >= "1984" AND f.release_date <= "2000" WITH d,f ORDER BY f.release_date ASC RETURN d, f`},
	}

	mutation := `MATCH (n:Film { filmId: {id} }) SET n.name = {name}`
	params := map[string]interface{}{"id": "m.0h_b6x1", "name": "Terminal"}
	driver := bolt.NewDriver()

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-Query", q.name), func(b *testing.B) {
			conn, err := driver.OpenNeo("bolt://localhost:7687")
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, _, err = conn.QueryNeoAll(q.query, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-Query-parallel", q.name), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				conn, err := driver.OpenNeo("bolt://localhost:7687")
				if err != nil {
					b.Fatal(err)
				}

				for pb.Next() {
					_, _, _, err = conn.QueryNeoAll(q.query, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-QueryAndMutation", q.name), func(b *testing.B) {
			conn, err := driver.OpenNeo("bolt://localhost:7687")
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, _, err = conn.QueryNeoAll(q.query, nil)
				if err != nil {
					b.Fatal(err)
				}
				_, err = conn.ExecNeo(mutation, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}

	for _, q := range queries {
		b.Run(fmt.Sprintf("%v-QueryAndMutation-parallel", q.name), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				conn, err := driver.OpenNeo("bolt://localhost:7687")
				if err != nil {
					b.Fatal(err)
				}

				for pb.Next() {
					_, _, _, err = conn.QueryNeoAll(q.query, nil)
					if err != nil {
						b.Fatal(err)
					}
					_, err = conn.ExecNeo(mutation, params)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
