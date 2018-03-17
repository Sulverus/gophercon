package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/goodsign/snowball"
)

const (
	filename  = "test.txt"
	algorithm = "en"
	encoding  = "UTF_8"
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
	go func() {
		fmt.Println("Profiling enabled on port 6060")
		http.ListenAndServe("localhost:6060", nil)
	}()
	// Создаем стеммер (для тестов используем snowball)
	stemmer, err := snowball.NewWordStemmer(algorithm, encoding)
	if err != nil {
		fmt.Printf("Init error: %v\n", err)
		return
	}
	defer stemmer.Close()

	// Создаем пустой индекс
	index := NewIndex(stemmer)
	fmt.Println(index)

	// Считываем докумнет
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

	start := time.Now()
	// Добавляем документы
	for _, doc := range corpus.Docs {
		index.Insert(doc.ID, doc.Title+" "+doc.Text, false)
	}
	fmt.Printf("Indexing finished. Resorting entries\n")
	index.Resort()

	//fmt.Println(index)
	fmt.Printf("Indexing time: %v\n", time.Since(start))

	total := 0.0
	for i := 0; i < 100; i++ {
		start = time.Now()
		//results := index.Search("aerodynamics: slipstream")
		results := index.Search("australian travellers")
		end := time.Since(start)
		total += float64(end) / 1000.0
		fmt.Printf("Search time: %v\n", end)
		if i == 99 {
			fmt.Printf("Relevant documents: %v\n", results)
		}
	}
	fmt.Printf("AVG Search time: %2f microseconds\n", total/100.0)
}
