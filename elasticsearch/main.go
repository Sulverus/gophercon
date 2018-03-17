package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	elastic "gopkg.in/olivere/elastic.v6"
	"io/ioutil"
)

const URL = "http://127.0.0.1:32769"

type Page struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type Input struct {
	Docs []Page `json:"docs"`
}

func main() {
	file, err := ioutil.ReadFile("test.txt")
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
		return
	}

	corpus := Input{}
	err = json.Unmarshal(file, &corpus)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}

	client, err := elastic.NewClient(
		elastic.SetURL(URL),
		elastic.SetSniff(false),
	)
	if err != nil {
		fmt.Printf("Elastic error: %v\n", err)
		return
	}

	info, code, err := client.Ping(URL).Do(context.Background())
	if err != nil {
		fmt.Printf("Elastic error: %v\n", err)
		return
	}
	fmt.Printf(
		"Elasticsearch returned with code %d and version %s\n",
		code, info.Version.Number,
	)

	esversion, err := client.ElasticsearchVersion(URL)
	if err != nil {
		fmt.Printf("Elastic error: %v\n", err)
		return
	}
	fmt.Printf("Elasticsearch version %s\n", esversion)

	exists, err := client.IndexExists("news").Do(context.Background())
	if err != nil {
		fmt.Printf("Elastic error: %v\n", err)
		return
	}
	if !exists {
		mapping := `
{
	"settings":{
		"number_of_shards":1,
		"number_of_replicas":0
	},
	"mappings":{
		"post":{
			"properties":{
				"id":{
				    "type":"long"
				},
				"title":{
					"type":"text",
					"store": true,
					"fielddata": true
				},
				"text":{
					"type":"text",
					"store": true,
					"fielddata": true
				}
			}
		}
	}
}
`
		_, err := client.CreateIndex("news").
			Body(mapping).
			Do(context.Background())
		if err != nil {
			fmt.Printf("Elastic error: %v\n", err)
			return
		}
	}

	for i, doc := range corpus.Docs {
		_, err := client.Index().
			Index("news").
			Type("post").
			Id(string(doc.ID)).
			BodyJson(doc).
			Do(context.Background())
		if err != nil {
			fmt.Printf("Error in doc %v", doc)
		}
		if i > 40000 {
			break
		}
	}
	fmt.Println("Loading done")

	_, err = client.Flush().Index("news").Do(context.Background())
	if err != nil {
		fmt.Printf("Elastic error: %v\n", err)
		return
	}

	sum := 0.0
	for i := 0; i < 100; i++ {
		termQuery := elastic.NewTermQuery(
			"title", "australian travellers",
		)
		searchResult, err := client.Search().
			Index("news").
			Query(termQuery).
			Sort("title", true).
			From(0).Size(10).
			Pretty(true).
			Do(context.Background())
		if err != nil {
			fmt.Printf("Elastic error: %v\n", err)
		}
		fmt.Printf(
			"Query took %d milliseconds\n",
			searchResult.TookInMillis,
		)
		sum += float64(searchResult.TookInMillis)
	}

	fmt.Printf("AVG Query took %.2f milliseconds\n", sum/100.0)
	client.DeleteIndex("news").Do(context.Background())
}
