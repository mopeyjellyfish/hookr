build:
	@echo "  >  Generating API"
	make msgpack
	@echo "  >  Building binary"
	tinygo build -o bin/simple.wasm -scheduler=none --no-debug -target=wasip1 -buildmode=c-shared main.go

## msgpack: build the msgpack
msgpack:
	cd ../api && go generate