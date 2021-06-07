SOURCE = $(wildcard *.go)
TAG ?= $(shell git describe --tags)
GOBUILD = go build -ldflags '-w'

ALL = \
	$(foreach suffix,linux mac mac-arm64 win.exe,\
		build/gr-64-$(suffix))

all: $(ALL)

clean:
	rm -f $(ALL)

# cram is a python app, so 'easy_install/pip install cram' to run tests
test:
	cram tests/main.t

# os is determined as thus: if variable of suffix exists, it's taken, if not, then
# suffix itself is taken
win.exe = GOOS=windows GOARCH=amd64
linux = GOOS=linux GOARCH=amd64
mac = GOOS=darwin GOARCH=amd64
mac-arm64 = GOOS=darwin GOARCH=arm64
build/gr-64-%: $(SOURCE)
	@mkdir -p $(@D)
	CGO_ENABLED=0 $($*) $(GOBUILD) -o $@

# NOTE: first push a tag, then make release!
release: $(ALL)
ifndef desc
	@echo "Run it as 'make release desc=tralala'"
else
	github-release release -u piranha -r goreplace -t "$(TAG)" -n "$(TAG)" --description "$(desc)"
	@for x in $(ALL); do \
		echo "Uploading $$x" && \
		github-release upload -u piranha \
                              -r goreplace \
                              -t $(TAG) \
                              -f "$$x" \
                              -n "$$(basename $$x)"; \
	done
endif
