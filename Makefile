SOURCE = $(wildcard *.go)
TAG = $(shell git describe --tags)
SUFFIX = win.exe linux osx
ALL = $(foreach suffix,$(SUFFIX),gr-$(TAG)-$(suffix)) $(foreach suffix,$(SUFFIX),gr-latest-$(suffix))

all: $(ALL)

clean:
	rm $(ALL)

# os is determined as thus: if variable of suffix exists, it's taken, if not, then
# suffix itself is taken
win.exe = windows
osx = darwin
gr-$(TAG)-%: $(SOURCE)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=amd64 go build -o $@

gr-latest-%: gr-$(TAG)-%
	ln -sf $< $@

upload: $(ALL)
ifndef UPLOAD_PATH
	@echo "Define UPLOAD_PATH to determine where files should be uploaded"
else
	rsync -l -P $(ALL) $(UPLOAD_PATH)
endif
