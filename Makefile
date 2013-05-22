SOURCE = $(wildcard *.go)
TAG = $(shell git describe --tags)

ALL = \
	$(foreach arch,32 64,\
	$(foreach tag,$(TAG) latest,\
	$(foreach suffix,win.exe osx linux,\
		gr-$(tag)-$(arch)-$(suffix))))

all: $(ALL)

clean:
	rm -f $(ALL)

# os is determined as thus: if variable of suffix exists, it's taken, if not, then
# suffix itself is taken
win.exe = windows
osx = darwin
gr-$(TAG)-64-%: $(SOURCE)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=amd64 go build -o $@

gr-$(TAG)-32-%: $(SOURCE)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=386 go build -o $@

gr-latest-%: gr-$(TAG)-%
	ln -sf $< $@

upload: $(ALL)
ifndef UPLOAD_PATH
	@echo "Define UPLOAD_PATH to determine where files should be uploaded"
else
	rsync -l -P $(ALL) $(UPLOAD_PATH)
endif
