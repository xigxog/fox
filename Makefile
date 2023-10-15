GIT_COMMIT = $(shell git log -n 1 --format="%h" -- ./)
GIT_REF := $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)

BUILD_DIR = bin
REL_DIR = release


.PHONY: all
all: clean build

.PHONY: build
build:
	go build -o $(BUILD_DIR)/fox .

.PHONY: docs
docs:
	go run main.go docs

.PHONY: release
release: clean
	# GIT_COMMIT=$(GIT_COMMIT) GIT_REF=$(GIT_REF) ./build.sh darwin $(BUILD_DIR) $(REL_DIR)
	GIT_COMMIT=$(GIT_COMMIT) GIT_REF=$(GIT_REF) ./build.sh linux $(BUILD_DIR) $(REL_DIR)
	# GIT_COMMIT=$(GIT_COMMIT) GIT_REF=$(GIT_REF) ./build.sh windows $(BUILD_DIR) $(REL_DIR)

.PHONY: clean
clean:
	go clean
	rm -rf ${BUILD_DIR} ${REL_DIR}

.PHONY: fmt
fmt:
	go fmt ./...