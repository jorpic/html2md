
all:
	go fmt ; go build

test: all
	go test && golint && go tool vet -all .
	
