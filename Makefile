all: build

.PHONY: build
build:
	@mkdir -p build
	@$(GO) build -o build/vm-spinner

.PHONY: clean
clean:
	@rm -fr build