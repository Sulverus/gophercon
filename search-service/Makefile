all:
	go build -o search-service *.go
profile:
	go tool pprof ./search-service http://localhost:6060/debug/pprof/profile
