package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile_CreatesNewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, WriteFile(path, []byte("hello"), 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestWriteFile_OverwritesExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, os.WriteFile(path, []byte("before"), 0o600))

	require.NoError(t, WriteFile(path, []byte("after"), 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "after", string(got))
}

func TestWriteFile_AppliesPerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits not enforced on windows")
	}
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, WriteFile(path, []byte("x"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestWriteFile_EmptyData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, WriteFile(path, nil, 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestWriteFile_LeavesNoTempOnSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, WriteFile(path, []byte("data"), 0o600))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("unexpected leftover temp file: %s", e.Name())
		}
	}
}

func TestWriteFile_EmptyPathRejected(t *testing.T) {
	t.Parallel()
	require.Error(t, WriteFile("", []byte("x"), 0o600))
}

func TestWriteFile_NonexistentDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "no-such-subdir", "out.txt")
	err := WriteFile(path, []byte("x"), 0o600)
	require.Error(t, err)
}

// On failure, the temp file should be cleaned up — never leak into the
// surrounding directory.
func TestWriteFile_NoTempLeakOnDirError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Use a path whose parent doesn't exist; CreateTemp fails before we'd
	// ever write data. The directory shouldn't end up with a stray file.
	path := filepath.Join(dir, "missing", "out.txt")
	_ = WriteFile(path, []byte("x"), 0o600)

	// The parent dir of "missing" is `dir`, which should still be empty.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		if e.Name() != "missing" { // missing was never created
			t.Fatalf("unexpected entry: %s", e.Name())
		}
	}
}

// Two concurrent writes to distinct paths must both succeed without
// interfering.
func TestWriteFile_ConcurrentDistinctPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const n = 16
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			p := filepath.Join(dir, "file-"+strings.Repeat("x", i+1)+".txt")
			data := []byte(strings.Repeat("z", i+1))
			require.NoError(t, WriteFile(p, data, 0o600))
		})
	}
	wg.Wait()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	regularFiles := 0
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
		regularFiles++
	}
	assert.Equal(t, n, regularFiles)
}

// Concurrent writes to the SAME path: each must produce a complete file
// (one of the values), never a partial / interleaved result.
func TestWriteFile_ConcurrentSamePath_AllOrNothing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "shared.txt")
	values := []string{
		strings.Repeat("A", 4096),
		strings.Repeat("B", 4096),
		strings.Repeat("C", 4096),
		strings.Repeat("D", 4096),
	}

	var wg sync.WaitGroup
	for _, v := range values {
		wg.Go(func() {
			require.NoError(t, WriteFile(path, []byte(v), 0o600))
		})
	}
	wg.Wait()

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	// File must equal exactly one of the candidate values; never a mix.
	assert.True(t, slices.Contains(values, string(got)),
		"final content was not any of the candidate writes")
}

func TestWriteFile_LargePayload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "big.bin")
	data := make([]byte, 1<<20) // 1 MiB
	for i := range data {
		data[i] = byte(i % 251)
	}
	require.NoError(t, WriteFile(path, data, 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, len(data), len(got))
	assert.Equal(t, data, got)
}

func TestWriteFile_OverwriteIsAtomicForReaders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("rename-over-open semantics differ on windows")
	}
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, os.WriteFile(path, []byte("OLD"), 0o600))

	// While another goroutine is repeatedly reading, swap the file
	// contents. The reader should always observe either the full old
	// content or the full new content.
	stop := make(chan struct{})
	bad := make(chan string, 1)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			b, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			s := string(b)
			if s != "OLD" && s != "NEW-CONTENT-LONGER" {
				select {
				case bad <- s:
				default:
				}
				return
			}
		}
	}()

	require.NoError(t, WriteFile(path, []byte("NEW-CONTENT-LONGER"), 0o600))
	close(stop)

	select {
	case s := <-bad:
		t.Fatalf("reader observed partial content: %q", s)
	default:
	}
}
