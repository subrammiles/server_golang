package iojson

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func ExampleCourse_marshal() {
	c := Course{ID: 1, Title: "Go", Price: 199950, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		internal: "hidden", Secret: "hidden too"}
	out, _ := json.Marshal(c)
	fmt.Println(string(out))
	// Output: {"id":1,"title":"Go","price":"1999.50","createdAt":"2026-01-02T03:04:05Z"}
}

func ExampleDecodeStrict() {
	var c Course
	err := DecodeStrict(strings.NewReader(`{"id":1,"titel":"typo"}`), &c)
	fmt.Println(err)
	err = DecodeStrict(strings.NewReader(`{"id":1,"price":"10.5"}`), &c)
	fmt.Println(err, c.Price)
	// Output:
	// json: unknown field "titel"
	// <nil> 1050
}

func ExampleCountLines() {
	fmt.Println(CountLines(strings.NewReader("one two\nthree\n\nfour five six\n")))
	// Output: 4 6 <nil>
}

func ExampleParseCSV() {
	rows, _ := ParseCSV(strings.NewReader("name,city\nAsha,Hyderabad\n\"Ravi, Jr\",Pune\n"))
	fmt.Println(rows[1]["name"], "|", rows[0]["city"])
	// Output: Ravi, Jr | Hyderabad
}

func ExampleEmbeddedCourses() {
	courses, err := EmbeddedCourses()
	fmt.Println(len(courses), courses[1].Title, courses[1].Price, courses[0].Tags, err)
	// Output: 2 Concurrency in Go 249950 [go basics] <nil>
}

func TestCompressAndHash(t *testing.T) {
	var compressed bytes.Buffer
	text := strings.Repeat("go is fun. ", 1000)
	sha, err := CompressAndHash(strings.NewReader(text), &compressed)
	if err != nil {
		t.Fatal(err)
	}
	if compressed.Len() >= len(text)/10 || len(sha) != 64 {
		t.Fatalf("compressed %d bytes, sha %q", compressed.Len(), sha)
	}
	r, _ := gzip.NewReader(&compressed)
	back, _ := io.ReadAll(r)
	if string(back) != text {
		t.Fatal("round trip failed")
	}
}

func TestWriteFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := WriteFileAtomically(path, []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomically(path, []byte(`{"v":2}`)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	entries, _ := os.ReadDir(filepath.Dir(path))
	if string(data) != `{"v":2}` || len(entries) != 1 {
		t.Fatalf("data %s, %d files (temp file left behind?)", data, len(entries))
	}
}

func TestMoneyRejectsNumbers(t *testing.T) {
	var m Money
	if err := json.Unmarshal([]byte(`19.99`), &m); err == nil {
		t.Fatal("want an error for a bare number")
	}
}
