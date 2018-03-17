package main

type Stemmer interface {
	Stem(word []byte) ([]byte, error)
}
