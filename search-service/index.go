package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	k1 = 2.0
	b  = 0.75
)

var (
	safeRe  = regexp.MustCompile(`(?i)[^a-zа-яё0-9 \-]+`)
	spaceRe = regexp.MustCompile(`(?i)[ \t]+`)
)

// Entry описывает вхождение слова в документ
type Entry struct {
	ID uint64
	TF uint
}

type EntryList []Entry

func (e EntryList) Len() int           { return len(e) }
func (e EntryList) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e EntryList) Less(i, j int) bool { return e[i].ID < e[j].ID }

type SearchResult struct {
	ID   uint64
	Rank float64
}

func (r SearchResult) String() string {
	return fmt.Sprintf("ID=%d, Rank=%f;", r.ID, r.Rank)
}

type ResultList []SearchResult

func (e ResultList) Len() int           { return len(e) }
func (e ResultList) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e ResultList) Less(i, j int) bool { return e[i].Rank > e[j].Rank }

// Index главная структура обратного индекса
type Index struct {
	docCount uint64
	stemmer  Stemmer
	storage  map[string]EntryList
	mapping  map[string]map[uint64]float64
	docsLen  map[uint64]float64
	avgLen   float64
	skipLen  int
}

// NewIndex возвращает проинициализированный обратный индекс
func NewIndex(stemmer Stemmer) *Index {
	return &Index{
		0, stemmer,
		map[string]EntryList{},
		map[string]map[uint64]float64{},
		map[uint64]float64{},
		0.0,
		0,
	}
}

// IDF возвращает значение inverted document frequency по формуле log(N/n(q))
func (i Index) IDF(term string) float64 {
	entries, ok := i.storage[term]
	if !ok || i.docCount == 0 {
		return 0.0
	}
	return math.Log(float64(i.docCount) / float64(len(entries)))
}

func (i *Index) buildSkips() {
	i.skipLen = int(math.Sqrt(float64(i.docCount)))
	fmt.Printf("Skiplist len=%d\n", i.skipLen)
}

// Добавляет вхождение в индекс
func (i *Index) insertEntry(term string, doc *Entry, resort bool) {
	elems, ok := i.storage[term]
	if !ok {
		elems = EntryList{}
	}
	elems = append(elems, *doc)

	if resort {
		sort.Sort(elems)
	}
	i.storage[term] = elems

	if _, ok := i.mapping[term]; !ok {
		i.mapping[term] = map[uint64]float64{}
	}
	i.mapping[term][doc.ID] = float64(doc.TF)
}

func (i *Index) Resort() {
	for key := range i.storage {
		elems := i.storage[key]
		sort.Sort(elems)
		i.storage[key] = elems
	}
	sum := float64(0)
	for _, freq := range i.docsLen {
		sum += freq
	}
	i.avgLen = sum / float64(i.docCount)
	i.buildSkips()
}

func (i Index) overlapLists(a EntryList, b EntryList) EntryList {
	p := 0
	j := 0

	result := EntryList{}

	for p < len(a) && j < len(b) {
		if a[p].ID == b[j].ID {
			result = append(result, a[p])
			p++
			j++
			continue
		}
		if a[p].ID > b[j].ID {
			// j++
			if i.skipLen > 0 && j+i.skipLen < len(b) && b[j+i.skipLen].ID <= a[p].ID {
				for j+i.skipLen < len(b) && b[j+i.skipLen].ID <= a[p].ID {
					j += i.skipLen
				}
			} else {
				j++
			}
		} else {
			//p++
			if i.skipLen > 0 && p+i.skipLen < len(a) && a[p+i.skipLen].ID <= b[j].ID {
				for p+i.skipLen < len(a) && a[p+i.skipLen].ID <= b[j].ID {
					p += i.skipLen
				}
			} else {
				p++
			}
		}

	}

	return result
}

func (i Index) Search(text string) ResultList {
	fmt.Printf("Query: '%s'\n", text)
	text = safeRe.ReplaceAllLiteralString(text, "")
	text = spaceRe.ReplaceAllLiteralString(text, " ")

	words := strings.Split(strings.ToLower(text), " ")
	terms := []string{}
	idfs := map[string]float64{}
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		wordStem, _ := i.stemmer.Stem([]byte(word))
		term := string(wordStem)
		terms = append(terms, term)
		idfs[term] = i.IDF(term)
	}

	fmt.Printf("Terms to find: %v\n", terms)
	result := EntryList{}

	for j, term := range terms {
		if j == 0 {
			result = i.storage[term]
			continue
		}
		result = i.overlapLists(result, i.storage[term])
	}

	ids := ResultList{}

	for _, entry := range result {
		rank := 0.0
		for _, term := range terms {
			tf := i.mapping[term][entry.ID]
			lenPart := i.docsLen[entry.ID] / i.avgLen

			bm25Top := tf * (k1 + 1.0)
			bm25Bottom := tf + k1*(1.0-b+b*lenPart)
			rank += (bm25Top / bm25Bottom) * idfs[term]
		}
		ids = append(
			ids, SearchResult{entry.ID, rank},
		)
	}
	sort.Sort(ids)
	return ids
}

// Insert добавляет документ в индекс
func (i *Index) Insert(docID uint64, text string, resort bool) {
	text = safeRe.ReplaceAllLiteralString(text, "")
	text = spaceRe.ReplaceAllLiteralString(text, " ")

	words := strings.Split(strings.ToLower(text), " ")
	zpf := map[string]uint{}

	for _, word := range words {
		if len(word) < 3 {
			continue
		}
		wordStem, err := i.stemmer.Stem([]byte(word))
		if nil != err {
			fmt.Printf("Stemming error: %v\n", err)
		} else {
			term := string(wordStem)
			_, ok := zpf[term]
			if !ok {
				zpf[term] = 0
			}
			zpf[term]++
		}
	}

	i.docsLen[docID] = 0
	for term, freq := range zpf {
		entry := Entry{docID, freq}
		i.insertEntry(term, &entry, resort)
		i.docsLen[docID] += float64(freq)
	}
	i.docCount++

	if resort {
		sum := float64(0)
		for _, freq := range i.docsLen {
			sum += freq
		}
		i.avgLen = sum / float64(i.docCount)
		i.buildSkips()
	}
}

// String приводит структуру к строке
func (i Index) String() string {
	return fmt.Sprintf("Inverted index. Documents count: %d", i.docCount)
}
