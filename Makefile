.PHONY: deps clean build

deps:
	go get -u ./...

clean: 
	rm -rf ./data-feeder/data-feeder
	
build:
	GOOS=linux GOARCH=amd64 go build -o data-feeder/data-feeder ./data-feeder