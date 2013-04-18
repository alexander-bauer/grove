package main

import (
	"crypto/rand"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
)

var tempDir string

type LogResponseWriter struct {
	Logf func(string, ...interface{})

	setStatus bool // True if status has already been set
}

func (w LogResponseWriter) Header() (h http.Header) {
	return nil
}

func (w LogResponseWriter) Write(b []byte) (n int, err error) {
	w.WriteHeader(http.StatusOK)
	return len(b), nil
}

func (w LogResponseWriter) WriteHeader(n int) {
	if !w.setStatus {
		w.Logf("Setting status to %d\n", n)
	} else {
		w.Logf("Tried to set status to %d, but is already set\n", n)
	}
}

func prepareRepository() (g *git, err error) {
	// We must set up a temporary git repository to serve. We use
	// ioutil.TempDir() to do this. ioutil will use the operating
	// system's temporary directory.
	tempDir, err = ioutil.TempDir("", "grove-testing")
	if err != nil {
		return
	}

	// Next, read 1Kb of random data using crypto/rand.
	buf := make([]byte, 1024)
	_, err = rand.Read(buf)
	if err != nil {
		return
	}

	// Then write the buf to a file.
	err = ioutil.WriteFile(path.Join(tempDir, "1Kb.bin"), buf, 0644)
	if err != nil {
		return
	}

	// Now create a *git object with the temporary directory.
	g = &git{
		Path: tempDir,
	}

	// Initialize the git directory with it, and commit and add the
	// random file.
	_, err = g.execute("init")
	if err != nil {
		return
	}
	_, err = g.execute("add", "1Kb.bin")
	if err != nil {
		return
	}
	_, err = g.execute("commit", "-m \"Initialize random data files\"")
	if err != nil {
		return
	}
	return
}

func removeTempDir() {
	os.RemoveAll(tempDir)
}

func BenchmarkMakeRaw(b *testing.B) {
	b.StopTimer()
	// Prepare the git repository with servable files.
	g, err := prepareRepository()
	if err != nil {
		b.Fatalf("Failed to prepare repository: %s", err)
		return
	}
	defer removeTempDir()

	// Create the fake ResponseWriter.
	w := LogResponseWriter{
		Logf: b.Logf,
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		err, _ = MakeRawPage(w, "1Kb.bin", "HEAD", g)
		if err != nil {
			b.Fatal(err)
			return
		}
	}
}

func BenchmarkMakeFile(b *testing.B) {
	b.StopTimer()
	g, err := prepareRepository()
	if err != nil {
		b.Fatalf("Failed to prepare repository: %s", err)
		return
	}
	defer removeTempDir()

	t, err = template.ParseFiles("res/templates/file.html")
	if err != nil {
		b.Fatalf("Failed to load template: %s", err)
		return
	}

	w := LogResponseWriter{
		Logf: b.Logf,
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		err, _ = MakeFilePage(w, &pageinfo{}, g, "HEAD", "1Kb.bin")
		if err != nil {
			b.Fatal(err)
			return
		}
	}
}
