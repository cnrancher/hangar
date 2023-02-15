TARGERS := build test ci
.PHONY: $(TARGERS) docker manifest clean help

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGERS): .dapper
	./.dapper $@

docker:
	@./scripts/docker.sh

clean:
	@./scripts/clean.sh

help:
	@echo "Usage:"
	@echo "    make build   -  Build 'hangar' executable files in 'build' folder"
	@echo "    make test    -  Run unit test"
	@echo "    make clean   -  Remove generated files"
	@echo "    make help    -  Show this message"
