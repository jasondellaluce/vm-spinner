all: vm-spinner

GO ?= go

.PHONY: vm-spinner
vm-spinner: main.go vagrant.go
	@mkdir -p build
	@$(GO) build -o build/vm-spinner *.go 

.PHONY: clean
clean:
	@rm -fr build