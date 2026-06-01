.PHONY: build install test clean scan

build:
	go build -o antislop .

install:
	go install .

test:
	go test ./... -count=1

clean:
	rm -f antislop

scan: build
	./antislop scan .
