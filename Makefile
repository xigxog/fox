REPO_ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: build
build:
	$(REPO_ROOT)hack/scripts/build.sh

.PHONY: release
release:
	$(REPO_ROOT)hack/scripts/release.sh

.PHONY: package
package:
	$(REPO_ROOT)hack/scripts/package.sh

.PHONY: image
image:
	$(REPO_ROOT)hack/scripts/image.sh

.PHONY: docs
docs:
	$(REPO_ROOT)hack/scripts/docs.sh

.PHONY: clean
clean:
	$(REPO_ROOT)hack/scripts/clean.sh

.PHONY: commit
commit:
	$(REPO_ROOT)hack/scripts/commit.sh
