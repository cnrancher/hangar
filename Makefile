.PHONY: help clean build install

image-tools:
	go build -o image-tools .

build: image-tools

install:
	go install .

clean:
	@./scripts/clean.sh

help:
	@echo "Usage:"
	@echo "    make build  -  build 'image-tools'"
	@echo "    make clean  -  remove generated files"