TARGETS := ci build test validate
.PHONY: $(TARGETS) $(TEST_TARGETS) validation-test clean help

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	@./.dapper $@

validation-test: .dapper
	@./.dapper -f Dockerfile.test.dapper

clean:
	@./scripts/clean.sh

help:
	@echo "Usage:"
	@echo "    make build           - Build 'hangar' executable files in 'bin' folder"
	@echo "    make test            - Run hangar unit test"
	@echo "    make validation-test - Run hangar validation test"
	@echo "    make clean           - Remove generated files"
	@echo "    make help            - Show this message"
