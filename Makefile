.PHONY: build install test clean

image-tools:
	go build -o image-tools .

build: image-tools

install:
	go install .

test:
	@./scripts/test.sh

clean:
	@./scripts/clean.sh

help:
	@echo "Usage:"
	@echo "    make build  -  build 'image-tools'"
	@echo "    make clean  -  remove generated files"