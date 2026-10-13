package testing06

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// TABLE-DRIVEN TEST: the standard Go style. One loop, many named cases, each run as a SUBTEST.
func TestSlugify(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"simple", "Hello World", "hello-world"},
		{"punctuation", "Hello, World!", "hello-world"},
		{"repeated separators", "a  --  b", "a-b"},
		{"leading and trailing", "  Go!  ", "go"},
		{"digits", "Go 1.25 released", "go-1-25-released"},
		{"unicode letters", "Café Crème", "café-crème"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // go test -run 'TestSlugify/digits' runs one case
			t.Parallel() // subtests may run in parallel
			if got := Slugify(tt.input); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want) // Errorf: report and continue
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := map[string]struct {
		password string
		want     error
	}{
		"too short": {"abc1", ErrTooShort},
		"no digit":  {"abcdefgh", ErrNoDigit},
		"valid":     {"abcdefg1", nil},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePassword(tt.password); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want) // Fatalf: report and STOP this test
			}
		})
	}
}

// ---- A FAKE implementation of the dependency ----

type fakeNotifier struct {
	sent []string
	err  error
}

func (f *fakeNotifier) Notify(user, message string) error {
	f.sent = append(f.sent, user+": "+message)
	return f.err
}

func TestSignupService(t *testing.T) {
	t.Run("sends a welcome message", func(t *testing.T) {
		fake := &fakeNotifier{}
		err := SignupService{Notifier: fake}.Register("asha", "secret123")
		assertNoError(t, err)
		assertEqual(t, len(fake.sent), 1)
		assertEqual(t, fake.sent[0], "asha: Welcome, asha!")
	})

	t.Run("invalid password: nothing is sent", func(t *testing.T) {
		fake := &fakeNotifier{}
		err := SignupService{Notifier: fake}.Register("asha", "short")
		if !errors.Is(err, ErrTooShort) {
			t.Fatalf("want ErrTooShort, got %v", err)
		}
		assertEqual(t, len(fake.sent), 0)
	})

	t.Run("notifier failure is returned", func(t *testing.T) {
		fake := &fakeNotifier{err: errors.New("smtp down")}
		err := SignupService{Notifier: fake}.Register("asha", "secret123")
		if err == nil || err.Error() != "smtp down" {
			t.Fatalf("got %v", err)
		}
	})
}

// HELPERS: t.Helper() makes failures point at the caller's line, not this one.
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// t.TempDir() and t.Cleanup(): files and resources removed automatically after the test.
func TestWritesReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte(Slugify("Monthly Report")), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { t.Log("cleanup runs after the test, even if it fails") })
	data, _ := os.ReadFile(path)
	assertEqual(t, string(data), "monthly-report")
}

// GOLDEN FILE: compare output with a file in testdata/ (ignored by the go tool).
// Regenerate after an intended change: go test ./06-testing -run Golden -update
var update = flag.Bool("update", false, "update golden files")

func TestSlugifyGolden(t *testing.T) {
	input, err := os.ReadFile("testdata/titles.txt")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	for _, line := range strings.Split(strings.TrimSpace(string(input)), "\n") {
		out.WriteString(Slugify(line) + "\n")
	}
	golden := filepath.Join("testdata", "titles.golden")
	if *update {
		os.WriteFile(golden, []byte(out.String()), 0o644)
	}
	want, _ := os.ReadFile(golden)
	assertEqual(t, out.String(), string(want))
}

// BENCHMARK: go test -bench=. -benchmem ./06-testing
func BenchmarkSlugify(b *testing.B) {
	for b.Loop() { // Go 1.24: the loop body is measured; setup outside it is not
		Slugify("The Quick Brown Fox Jumps Over The Lazy Dog, 2025 Edition!")
	}
}

// FUZZ TEST: Go generates random inputs looking for crashes or broken properties.
// Plain "go test" only runs the seed corpus; fuzz with: go test -fuzz=FuzzSlugify -fuzztime=10s ./06-testing
func FuzzSlugify(f *testing.F) {
	for _, seed := range []string{"Hello World", "", "---", "Ünïcödé 123"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		slug := Slugify(input)
		// PROPERTIES that must hold for every input:
		if !utf8.ValidString(input) {
			return
		}
		if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") || strings.Contains(slug, "--") {
			t.Errorf("bad dashes in %q (input %q)", slug, input)
		}
		if Slugify(slug) != slug {
			t.Errorf("not idempotent: %q -> %q", slug, Slugify(slug))
		}
	})
}
