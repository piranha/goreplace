SOURCE = $(wildcard *.go)
TAG = $(shell git describe --tags)

ALL = \
	$(foreach arch,32 64,\
	$(foreach tag,$(TAG) latest,\
	$(foreach suffix,win.exe osx linux,\
		build/gr-$(tag)-$(arch)-$(suffix))))

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
build/gr-$(TAG)-64-%: $(SOURCE)
	@mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=amd64 go build -o $@

build/gr-$(TAG)-32-%: $(SOURCE)
	@mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=386 go build -o $@

build/gr-latest-%: build/gr-$(TAG)-%
	@mkdir -p $(@D)
	ln -sf $< $@

upload: $(ALL)
ifndef UPLOAD_PATH
	@echo "Define UPLOAD_PATH to determine where files should be uploaded"
else
	rsync -l -P $(ALL) $(UPLOAD_PATH)
endif
