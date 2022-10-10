GOFILES = $(shell find . -name '*.go')

.PHONY: build clean yamux
build: yamux

yamux: bin/yamux

bin/yamux: $(GOFILES)
	@mkdir -p bin
	go build -trimpath -o ./bin/yamux .

clean:
	rm -r bin
