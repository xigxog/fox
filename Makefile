build_dir = bin
rel_dir = release


.PHONY: all
all: clean build

.PHONY: build
build:
	go build -o $(build_dir)/fox .

.PHONY: docs
docs:
	go run main.go docs

.PHONY: release
release: clean
	./build.sh darwin $(build_dir) $(rel_dir)
	./build.sh linux $(build_dir) $(rel_dir)
	./build.sh windows $(build_dir) $(rel_dir)

.PHONY: clean
clean:
	go clean
	rm -rf ${build_dir} ${rel_dir}

.PHONY: fmt
fmt:
	go fmt ./...