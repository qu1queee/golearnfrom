BIN := glean
CMD := ./cmd/glean

.PHONY: build clean

build:
	go build -o $(BIN) $(CMD)

clean:
	rm -f $(BIN)
