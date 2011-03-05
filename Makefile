include $(GOROOT)/src/Make.inc

.SUFFIXES: .go .$O

TARG=gr
MAIN=goreplace
GOFILES=highlight.go goreplace.go

all: $(TARG)

run: all
	./$(TARG) main

### boilerplace

OBJS=$(GOFILES:.go=.$O)

$(TARG): $(OBJS)
	$(LD) -o $(TARG) $(MAIN).$O

clean:
	rm -f $(OBJS) $(TARG)

.go.$O:
	$(GC) $<
