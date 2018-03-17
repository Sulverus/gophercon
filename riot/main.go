package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-ego/riot"
	"github.com/go-ego/riot/types"
)

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

	searcher := riot.Engine{}
	searcher.Init(types.EngineOpts{
		Using:                     4,
		NumShards:                 1,
		NumIndexerThreadsPerShard: 1,
		NumRankerThreadsPerShard:  1,
		StorageShards:             1,
		StorageEngine:             "bg",
		NotUsingGse:               true})
	defer searcher.Close()

	start := time.Now()
	for _, doc := range corpus.Docs {
		searcher.IndexDoc(doc.ID, types.DocIndexData{
			Content: doc.Title + " " + doc.Text,
		})
	}
	searcher.Flush()
	fmt.Printf("Indexing finished. Resorting entries\n")
	fmt.Printf("Indexing time: %v\n", time.Since(start))

	total := 0.0
	for i := 0; i < 100; i++ {
		start = time.Now()
		results := searcher.Search(types.SearchReq{
			Text: "australian travellers",
		})
		end := time.Since(start)
		fmt.Printf("Search time: %v\n", end)
		total += float64(end) / 1000.0
		if i == 99 {
			fmt.Printf("Relevant documents: %v\n", results)
		}
	}
	fmt.Printf("AVG Search time: %2f microseconds\n", total/100.0)
}
