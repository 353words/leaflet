src_files = $(wildcard *.go) index.html map.html map.js ./testdata/Morning_Walk.gpx

all:
	@echo Please pick a target, should be one of:
	@grep -E '^[a-z]+:' Makefile | awk -NF : '{print "    " $$1}'

test:
	staticcheck .
	gosec .
	go test -v

html:
	for src in $(src_files) ; do \
		echo $$src ; \
		test -f $$src && ../blog_fmt.sh $$src ;\
	done

deps:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest


size:
	cloc httpd.go gpx.go index.html map.html map.js
