test:
	go test -v -coverprofile=coverage.out -coverpkg=./... ./...

bench:
	go test -test.bench=. ./...

build:
	go build

install:
	go install

build-vscode:
	$(MAKE) -C editors/vscode build
	
