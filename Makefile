# ==================================================================================== #
# HELPERS
# ==================================================================================== #


## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run the cmd/api application
.PHONY: run
run:
	@echo 'Runing ......'
	go run ./...


# ==================================================================================== #
# BUILD
# ==================================================================================== #

current_time = $(shell date --iso-8601=seconds)
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.BuildTime=${current_time} -X main.Version=${git_description}'

## build: build the application
.PHONY: build
build:
	@echo 'Building ......'
	go build -ldflags=${linker_flags} -o=./bin/eattachs ./...

.PHONY: build/win
build/win:
	@echo 'Building ......'
	go build -ldflags='${linker_flags}' -o=./bin/eattachs ./...
	GOOS=windows GOARCH=amd64 go build -ldflags='${linker_flags}' -o=./bin/eattachs_win ./...