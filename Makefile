.PHONY: build install test release clean

build:
	go build -o image-tools .

install:
	go install .

test:
	@./scripts/test.sh

release:
	@./scripts/release.sh

clean:
	@./scripts/clean.sh

help:
	@echo "Usage:"
	@echo "    make build  -  build 'image-tools'"
	@echo "    make clean  -  remove generated files"