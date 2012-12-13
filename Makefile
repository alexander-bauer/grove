GOCOMPILER=go build
COMPILERFLAGS=

.DEFAULT_TARGET : grove
grove:
	$(GOCOMPILER) $(COMPILERFLAGS)

.PHONY : clean
clean:
	rm -f grove