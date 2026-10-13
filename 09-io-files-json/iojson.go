// Package iojson covers io.Reader/Writer composition, files, bufio, JSON, CSV and embedded files.
package iojson

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Course shows JSON STRUCT TAGS: field names in JSON, omitting empty values, and hiding fields.
type Course struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags,omitempty"` // left out when empty
	Price     Money     `json:"price"`          // custom JSON format (see below)
	CreatedAt time.Time `json:"createdAt"`      // RFC 3339 automatically
	internal  string    // unexported fields are never serialised
	Secret    string    `json:"-"` // explicitly excluded
}

// Money stores paise (integer) to avoid float rounding errors, but is written as "1999.00" in JSON.
type Money int64

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%d.%02d", m/100, m%100))
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("money must be a string like \"19.99\": %w", err)
	}
	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid money %q", s)
	}
	f, _ := strconv.ParseInt((frac + "00")[:2], 10, 64)
	*m = Money(w*100 + f)
	return nil
}

// DecodeStrict rejects unknown fields: a typo like "titel" becomes an error instead of being silently ignored.
func DecodeStrict(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// CountLines reads ANY io.Reader line by line with bufio.Scanner: files, network bodies, strings...
func CountLines(r io.Reader) (lines, words int, err error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		lines++
		words += len(strings.Fields(sc.Text()))
	}
	return lines, words, sc.Err()
}

// CompressAndHash shows composition: data flows src -> gzip -> (file AND hash) in one pass, without loading it all.
func CompressAndHash(src io.Reader, dst io.Writer) (sha string, err error) {
	h := sha256.New()
	gz := gzip.NewWriter(io.MultiWriter(dst, h)) // write to two places at once
	if _, err := io.Copy(gz, src); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil { // Close flushes the gzip footer: never skip it
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteFileAtomically writes to a temp file then renames it: readers never see a half-written file.
func WriteFileAtomically(path string, data []byte) (err error) {
	tmp, err := os.CreateTemp(dirOf(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return "."
}

// ParseCSV reads a CSV with a header row into maps.
func ParseCSV(r io.Reader) ([]map[string]string, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	header := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		m := make(map[string]string, len(header))
		for i, col := range header {
			m[col] = row[i]
		}
		out = append(out, m)
	}
	return out, nil
}

// go:embed puts files INSIDE the compiled binary: templates, SQL migrations, static assets.
// One binary to deploy, nothing to lose.
//
//go:embed testdata/courses.json
var embedded embed.FS

func EmbeddedCourses() ([]Course, error) {
	f, err := embedded.Open("testdata/courses.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var courses []Course
	return courses, json.NewDecoder(f).Decode(&courses)
}
