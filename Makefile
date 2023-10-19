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
	./build.sh darwin $(BUILD_DIR) $(REL_DIR)
	./build.sh linux $(BUILD_DIR) $(REL_DIR)
	./build.sh windows $(BUILD_DIR) $(REL_DIR)

.PHONY: clean
clean:
	go clean
	rm -rf ${BUILD_DIR} ${REL_DIR}

.PHONY: fmt
fmt:
	go fmt ./...