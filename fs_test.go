// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dep

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang/dep/internal/test"
)

func TestCopyDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	srcdir := filepath.Join(dir, "src")
	if err = os.MkdirAll(srcdir, 0755); err != nil {
		t.Fatal(err)
	}

	files := []struct {
		path     string
		contents string
		fi       os.FileInfo
	}{
		{path: "myfile", contents: "hello world"},
		{path: filepath.Join("subdir", "file"), contents: "subdir file"},
	}

	// Create structure indicated in 'files'
	for i, file := range files {
		fn := filepath.Join(srcdir, file.path)
		dn := filepath.Dir(fn)
		if err = os.MkdirAll(dn, 0755); err != nil {
			t.Fatal(err)
		}

		fh, err := os.Create(fn)
		if err != nil {
			t.Fatal(err)
		}

		if _, err = fh.Write([]byte(file.contents)); err != nil {
			t.Fatal(err)
		}
		fh.Close()

		files[i].fi, err = os.Stat(fn)
		if err != nil {
			t.Fatal(err)
		}
	}

	destdir := filepath.Join(dir, "dest")
	if err = CopyDir(srcdir, destdir); err != nil {
		t.Fatal(err)
	}

	// Compare copy against structure indicated in 'files'
	for _, file := range files {
		fn := filepath.Join(srcdir, file.path)
		dn := filepath.Dir(fn)
		dirOK, err := IsDir(dn)
		if err != nil {
			t.Fatal(err)
		}
		if !dirOK {
			t.Fatalf("expected %s to be a directory", dn)
		}

		got, err := ioutil.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}

		if file.contents != string(got) {
			t.Fatalf("expected: %s, got: %s", file.contents, string(got))
		}

		gotinfo, err := os.Stat(fn)
		if err != nil {
			t.Fatal(err)
		}

		if file.fi.Mode() != gotinfo.Mode() {
			t.Fatalf("expected %s: %#v\n to be the same mode as %s: %#v",
				file.path, file.fi.Mode(), fn, gotinfo.Mode())
		}
	}
}

func TestCopyDirFailSrc(t *testing.T) {
	if runtime.GOOS == "windows" {
		// XXX: setting permissions works differently in
		// Microsoft Windows. Skipping this this until a
		// compatible implementation is provided.
		t.Skip("skipping on windows")
	}

	var srcdir, dstdir string

	err, cleanup := setupInaccesibleDir(func(dir string) (err error) {
		srcdir = filepath.Join(dir, "src")
		return os.MkdirAll(srcdir, 0755)
	})

	defer cleanup()

	if err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dstdir = filepath.Join(dir, "dst")
	if err = CopyDir(srcdir, dstdir); err == nil {
		t.Fatalf("expected error for CopyDir(%s, %s), got none", srcdir, dstdir)
	}
}

func TestCopyDirFailDst(t *testing.T) {
	if runtime.GOOS == "windows" {
		// XXX: setting permissions works differently in
		// Microsoft Windows. Skipping this this until a
		// compatible implementation is provided.
		t.Skip("skipping on windows")
	}

	var srcdir, dstdir string

	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	srcdir = filepath.Join(dir, "src")
	if err = os.MkdirAll(srcdir, 0755); err != nil {
		t.Fatal(err)
	}

	err, cleanup := setupInaccesibleDir(func(dir string) error {
		dstdir = filepath.Join(dir, "dst")
		return nil
	})

	defer cleanup()

	if err != nil {
		t.Fatal(err)
	}

	if err = CopyDir(srcdir, dstdir); err == nil {
		t.Fatalf("expected error for CopyDir(%s, %s), got none", srcdir, dstdir)
	}
}

func TestCopyDirFailOpen(t *testing.T) {
	if runtime.GOOS == "windows" {
		// XXX: setting permissions works differently in
		// Microsoft Windows. os.Chmod(..., 0222) below is not
		// enough for the file to be readonly, and os.Chmod(...,
		// 0000) returns an invalid argument error. Skipping
		// this this until a compatible implementation is
		// provided.
		t.Skip("skipping on windows")
	}

	var srcdir, dstdir string

	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	srcdir = filepath.Join(dir, "src")
	if err = os.MkdirAll(srcdir, 0755); err != nil {
		t.Fatal(err)
	}

	srcfn := filepath.Join(srcdir, "file")
	srcf, err := os.Create(srcfn)
	if err != nil {
		t.Fatal(err)
	}
	srcf.Close()

	// setup source file so that it cannot be read
	if err = os.Chmod(srcfn, 0222); err != nil {
		t.Fatal(err)
	}

	dstdir = filepath.Join(dir, "dst")

	if err = CopyDir(srcdir, dstdir); err == nil {
		t.Fatalf("expected error for CopyDir(%s, %s), got none", srcdir, dstdir)
	}
}

func TestCopyFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	srcf, err := os.Create(filepath.Join(dir, "srcfile"))
	if err != nil {
		t.Fatal(err)
	}

	want := "hello world"
	if _, err := srcf.Write([]byte(want)); err != nil {
		t.Fatal(err)
	}
	srcf.Close()

	destf := filepath.Join(dir, "destf")
	if err := CopyFile(srcf.Name(), destf); err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadFile(destf)
	if err != nil {
		t.Fatal(err)
	}

	if want != string(got) {
		t.Fatalf("expected: %s, got: %s", want, string(got))
	}

	wantinfo, err := os.Stat(srcf.Name())
	if err != nil {
		t.Fatal(err)
	}

	gotinfo, err := os.Stat(destf)
	if err != nil {
		t.Fatal(err)
	}

	if wantinfo.Mode() != gotinfo.Mode() {
		t.Fatalf("expected %s: %#v\n to be the same mode as %s: %#v", srcf.Name(), wantinfo.Mode(), destf, gotinfo.Mode())
	}
}

func TestCopyFileFail(t *testing.T) {
	if runtime.GOOS == "windows" {
		// XXX: setting permissions works differently in
		// Microsoft Windows. Skipping this this until a
		// compatible implementation is provided.
		t.Skip("skipping on windows")
	}

	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	srcf, err := os.Create(filepath.Join(dir, "srcfile"))
	if err != nil {
		t.Fatal(err)
	}
	srcf.Close()

	var dstdir string

	err, cleanup := setupInaccesibleDir(func(dir string) error {
		dstdir = filepath.Join(dir, "dir")
		return os.Mkdir(dstdir, 0777)
	})

	defer cleanup()

	if err != nil {
		t.Fatal(err)
	}

	fn := filepath.Join(dstdir, "file")
	if err := CopyFile(srcf.Name(), fn); err == nil {
		t.Fatalf("expected error for %s, got none", fn)
	}
}

// setupInaccesibleDir creates a temporary location with a single
// directory in it, in such a way that that directory is not accessible
// after this function returns.
//
// The provided operation op is called with the directory as argument,
// so that it can create files or other test artifacts.
//
// This function returns a nil error on success, and a cleanup function
// that removes all the temporary files this function creates. It is
// the caller's responsability to call this function before the test is
// done running, whether there's an error or not.
func setupInaccesibleDir(op func(dir string) error) (err error, cleanup func()) {
	cleanup = func() {}

	dir, err := ioutil.TempDir("", "dep")
	if err != nil {
		return err, cleanup
	}

	subdir := filepath.Join(dir, "dir")

	cleanup = func() {
		os.Chmod(subdir, 0777)
		os.RemoveAll(dir)
	}

	if err = os.Mkdir(subdir, 0777); err != nil {
		return err, cleanup
	}

	if err = op(subdir); err != nil {
		return err, cleanup
	}

	if err = os.Chmod(subdir, 0666); err != nil {
		return err, cleanup
	}

	return err, cleanup
}

func TestIsRegular(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var fn string

	err, cleanup := setupInaccesibleDir(func(dir string) error {
		fn = filepath.Join(dir, "file")
		fh, err := os.Create(fn)
		if err != nil {
			return err
		}

		return fh.Close()
	})

	defer cleanup()

	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		exists bool
		err    bool
	}{
		wd: {false, true},
		filepath.Join(wd, "testdata"):                       {false, true},
		filepath.Join(wd, "cmd", "dep", "main.go"):          {true, false},
		filepath.Join(wd, "this_file_does_not_exist.thing"): {false, false},
		fn: {false, true},
	}

	if runtime.GOOS == "windows" {
		// This test doesn't work on Microsoft Windows because
		// of the differences in how file permissions are
		// implemented. For this to work, the directory where
		// the file exists should be inaccessible.
		delete(tests, fn)
	}

	for f, want := range tests {
		got, err := IsRegular(f)
		if err != nil {
			if want.exists != got {
				t.Fatalf("expected %t for %s, got %t", want.exists, f, got)
			}
			if !want.err {
				t.Fatalf("expected no error, got %v", err)
			}
		} else {
			if want.err {
				t.Fatalf("expected error for %s, got none", f)
			}
		}

		if got != want.exists {
			t.Fatalf("expected %t for %s, got %t", want, f, got)
		}
	}

}

func TestIsDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var dn string

	err, cleanup := setupInaccesibleDir(func(dir string) error {
		dn = filepath.Join(dir, "dir")
		return os.Mkdir(dn, 0777)
	})

	defer cleanup()

	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		exists bool
		err    bool
	}{
		wd: {true, false},
		filepath.Join(wd, "testdata"):                       {true, false},
		filepath.Join(wd, "main.go"):                        {false, false},
		filepath.Join(wd, "this_file_does_not_exist.thing"): {false, true},
		dn: {false, true},
	}

	if runtime.GOOS == "windows" {
		// This test doesn't work on Microsoft Windows because
		// of the differences in how file permissions are
		// implemented. For this to work, the directory where
		// the directory exists should be inaccessible.
		delete(tests, dn)
	}

	for f, want := range tests {
		got, err := IsDir(f)
		if err != nil {
			if want.exists != got {
				t.Fatalf("expected %t for %s, got %t", want.exists, f, got)
			}
			if !want.err {
				t.Fatalf("expected no error, got %v", err)
			}
		}

		if got != want.exists {
			t.Fatalf("expected %t for %s, got %t", want.exists, f, got)
		}
	}

}

func TestIsEmpty(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	h := test.NewHelper(t)
	defer h.Cleanup()

	h.TempDir("empty")
	tests := map[string]string{
		wd:                                                  "true",
		"testdata":                                          "true",
		filepath.Join(wd, "fs.go"):                          "err",
		filepath.Join(wd, "this_file_does_not_exist.thing"): "false",
		h.Path("empty"):                                     "false",
	}

	for f, want := range tests {
		empty, err := IsNonEmptyDir(f)
		if want == "err" {
			if err == nil {
				t.Fatalf("Wanted an error for %v, but it was nil", f)
			}
			if empty {
				t.Fatalf("Wanted false with error for %v, but got true", f)
			}
		} else if err != nil {
			t.Fatalf("Wanted no error for %v, got %v", f, err)
		}

		if want == "true" && !empty {
			t.Fatalf("Wanted true for %v, but got false", f)
		}

		if want == "false" && empty {
			t.Fatalf("Wanted false for %v, but got true", f)
		}
	}
}
