DCR=docker-compose run --rm
.PHONY: test

all: release

test:
	$(DCR) go-test
