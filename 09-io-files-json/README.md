# 09: I/O, files, JSON and CSV

## io.Reader and io.Writer
Two one-method interfaces connect everything: files, network connections, HTTP bodies, buffers, gzip, hashes.
Write functions that accept `io.Reader`/`io.Writer` and they work with all of them, in tests too (`strings.NewReader`, `bytes.Buffer`).
- `io.Copy`, `io.MultiWriter`, `io.ReadAll` (careful with big inputs), `io.LimitReader`.
- `CompressAndHash`: data flows source -> gzip -> file + sha256 in **one streaming pass**.

## Concepts
- **Files**: `os.ReadFile`/`WriteFile` for small files. `os.Open` + `defer f.Close()` for streaming.
  **Atomic writes** (temp file + rename) so readers never see half a file.
- **`bufio.Scanner`**: read line by line. `bufio.Writer` batches small writes (remember `Flush`).
- **JSON** (`encoding/json`): struct tags (`json:"name,omitempty"`, `json:"-"`), `Marshal`/`Unmarshal`,
  streaming `Decoder`/`Encoder`, `DisallowUnknownFields` to catch typos, and custom `MarshalJSON`/`UnmarshalJSON`
  (`Money` stores paise as an integer and avoids float errors).
- **CSV** with `encoding/csv` (handles quotes and commas inside fields).
- **`//go:embed`**: ship files (templates, migrations, assets) inside the binary.

## Run
```bash
go test -v ./09-io-files-json
```
