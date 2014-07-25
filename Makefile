SOURCE = $(wildcard *.go)
TAG = $(shell git describe --tags)
GOBUILD = go build -ldflags '-w'

ALL = \
	$(foreach arch,32 64,\
	$(foreach suffix,win.exe osx linux,\
		build/gr-$(arch)-$(suffix)))

all: $(ALL)

clean:
	rm -f $(ALL)

# cram is a python app, so 'easy_install/pip install cram' to run tests
test:
	cram tests/main.t

# os is determined as thus: if variable of suffix exists, it's taken, if not, then
# suffix itself is taken
win.exe = windows
osx = darwin
build/gr-64-%: $(SOURCE)
	@mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=amd64 $(GOBUILD) -o $@

build/gr-32-%: $(SOURCE)
	@mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=386 $(GOBUILD) -o $@

release: $(ALL)
ifndef desc
	@echo "Run it as 'make release desc=tralala'"
else
	github-release release -u piranha -r goreplace -t "$(TAG)" -n "$(TAG)" --description "$(desc)"
	@for x in $(ALL); do \
		github-release upload -u piranha \
                              -r goreplace \
                              -t $(TAG) \
                              -f "$$x" \
                              -n "$$(basename $$x)" \
		&& echo "Uploaded $$x"; \
	done
endif
