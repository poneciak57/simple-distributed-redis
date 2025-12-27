
export GOEXPERIMENT=rangefunc

build:
	mkdir -p bin
	go build -o bin/app main.go

main: build
	go run main.go

test:
	go test ./... -v

clean:
	rm -rf bin/