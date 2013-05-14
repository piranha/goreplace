SOURCE = $(wildcard *.go)
TAG = $(shell git describe --tags)
ALL = $(foreach suffix,win.exe linux osx,gr-$(TAG)-$(suffix))

all: $(ALL)

clean:
	rm $(ALL)

# os is determined as thus: if variable of suffix exists, it's taken, if not, then
# suffix itself is taken
win.exe = windows
osx = darwin
gr-$(TAG)-%: $(SOURCE)
	CGO_ENABLED=0 GOOS=$(firstword $($*) $*) GOARCH=amd64 go build -o $@

upload: $(ALL)
	rsync -P $(ALL) $(UPLOAD_PATH)
