include $(GOROOT)/src/Make.inc

.SUFFIXES: .go .$O

TARG = gr
MAIN = goreplace
GOFILES = ignore.go goreplace.go

all: $(TARG)

run: all
	./$(TARG) main

### boilerplace

OBJS=$(GOFILES:.go=.$O)

$(TARG): $(OBJS)
	$(LD) -o $(TARG) $(MAIN).$O

prod: $(OBJS)
	$(LD) -s -o $(TARG) $(MAIN).$O

clean:
	rm -f $(OBJS) $(TARG)

%.$O: %.go
	$(GC) $<
