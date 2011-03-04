include $(GOROOT)/src/Make.inc

TARG=goreplace
GOFILES=goreplace.go
MAIN=goreplace.go

gr: package
	$(GC) -I_obj $(MAIN)
	$(LD) -L_obj -o $@ $(MAIN:%.go=%).$O
	@echo "Done. Executable is: $@"

include $(GOROOT)/src/Make.pkg
