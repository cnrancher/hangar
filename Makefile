.PHONY: build install test clean

build:
	go build -o image-tools .

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