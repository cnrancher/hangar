TARGETS := build test ci build-all
TEST_TARGETS := test_convert-list test_generate-list \
	test_help test_load test_save test_mirror test_version \
	test_mirror-validate test_load-validate \
	test_sync test_compress test_decompress test_all
.PHONY: $(TARGETS) $(TEST_TARGETS) docker manifest clean help

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	@./.dapper $@

$(TEST_TARGETS): .dapper
	@./.dapper --file Dockerfile-test.dapper $@

docker:
	@./scripts/docker.sh

clean:
	@./scripts/clean.sh

help:
	@echo "Usage:"
	@echo "    make build          - Build 'hangar' executable files in 'build' folder"
	@echo "    make test           - Run unit test"
	@echo "    make test_[COMMAND] - Run automation test on specific Hangar command."
	@echo "    make test_all       - Run automation test on all Hangar commands."
	@echo "    make clean          - Remove generated files"
	@echo "    make help           - Show this message"
