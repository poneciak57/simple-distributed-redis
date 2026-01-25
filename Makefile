
export GOEXPERIMENT=rangefunc

src/raft/pb:
	@mkdir -p src/raft/pb

proto: src/raft/pb
	export PATH=$$(go env GOPATH)/bin:$$PATH; \
	cd proto && protoc --go_out=../src/raft/pb --go_opt=paths=source_relative \
    --go-grpc_out=../src/raft/pb --go-grpc_opt=paths=source_relative \
    raft.proto

build: proto
	mkdir -p bin
	go build -o bin/app main.go

main: build
	go run main.go

test: proto
	go test -race -v ./tests

clean:
	rm -rf bin/