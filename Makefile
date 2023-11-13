src_files = $(wildcard *.go) $(wildcard static/*.html) ./testdata/Morning_Walk.gpx

test:
	staticcheck .
	gosec .
	go test -v

html:
	for src in $(src_files) ; do \
		../blog_fmt.sh $$src ;\
	done

