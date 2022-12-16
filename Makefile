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
	@echo "    make build   -  Build 'image-tools'"
	@echo "    make install -  Install image-tools into \$$GOPATH/bin"
	@echo "    make test    -  Run unit test"
	@echo "    make release -  Build all platform and architecture binaried in 'release' folder"
	@echo "    make clean   -  Remove generated files"
	@echo "    make help    -  Show this message"
