SOURCE = $(wildcard *.go)
TAG = $(shell git describe --tags)
ALL = $(foreach os,windows linux darwin,gr-$(TAG)-$(os))

all: $(ALL)

clean:
	rm $(ALL)

upload: $(addprefix upload-,$(ALL)) $(ALL)

upload-%: %
	github-upload.py $*

gr-$(TAG)-%: $(SOURCE)
	CGO_ENABLED=0 GOOS=$* GOARCH=amd64 go build -o $@ goreplace
