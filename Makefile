src_files = $(wildcard *.go) $(wildcard static/*.html) ./testdata/Morning_Walk.gpx

all:
	@echo Please pick a target, should be one of:
	@grep -E '^[a-z]+:' Makefile | awk -NF : '{print "    " $$1}'

test:
	staticcheck .
	gosec .
	go test -v

html:
	for src in $(src_files) ; do \
		../blog_fmt.sh $$src ;\
	done

deps:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
