include $(GOROOT)/src/Make.inc

TARG=gr
GOFILES=goreplace.go

main: all

run: main
	./gr main

include $(GOROOT)/src/Make.cmd
