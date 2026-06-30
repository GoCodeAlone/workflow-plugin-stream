.PHONY: build test install-local clean

build:
	go build -o workflow-plugin-stream ./cmd/workflow-plugin-stream

test:
	go test ./...

install-local: build
	wfctl plugin install --local .

clean:
	rm -f workflow-plugin-stream
