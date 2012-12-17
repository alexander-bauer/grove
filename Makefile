GOCOMPILER=go build
COMPILERFLAGS=

.DEFAULT_TARGET : grove
grove:
	$(GOCOMPILER) $(COMPILERFLAGS)

clean:
	rm -f grove
	rm -f grove-*.tar.gz

pack: grove FORCE
	git describe --exact-match --abbrev=0
	tar -czf grove-$(shell git describe --exact-match --match v* --abbrev=0 HEAD).tar.gz README.md grove res/

FORCE:

.PHONY : clean pack