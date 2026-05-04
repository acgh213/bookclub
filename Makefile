.PHONY: build clean stop start restart test

build:
	go build -o bookclub ./cmd/srv

clean:
	rm -f bookclub

test:
	go test ./...
