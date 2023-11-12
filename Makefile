src_files = $(wildcard *.go) $(wildcard static/*.html) Morning_Walk.gpx
html:
	for src in $(src_files) ; do \
		../blog_fmt.sh $$src ;\
	done

