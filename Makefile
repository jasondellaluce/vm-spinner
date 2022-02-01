all: vm-spinner

GO ?= go

.PHONY: vm-spinner
vm-spinner:
	@mkdir -p build
	@$(GO) build -o build/vm-spinner cmd/vm-spinner/main.go

.PHONY: clean
clean:
	@rm -fr build
