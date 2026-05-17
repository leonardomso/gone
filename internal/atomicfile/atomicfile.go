// Package atomicfile provides atomic file write primitives. The goal is that
// after a successful WriteFile call, readers either see the full new content
// or the full previous content — never a partial / truncated file caused by a
// crash, OS reboot, disk-full, or SIGKILL halfway through the write.
package atomicfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFile atomically writes data to the file named by path. It writes to a
// temp file in the same directory (so the final rename is atomic on the same
// filesystem), fsyncs the temp file, then renames it over path. On any error
// the temp file is removed.
//
// perm is applied to the final file via chmod after creation; on Unix the
// resulting file ignores umask, matching os.WriteFile semantics.
func WriteFile(path string, data []byte, perm fs.FileMode) (retErr error) {
	if path == "" {
		return errors.New("atomicfile: empty path")
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// On any failure past this point, remove the leftover temp file. The
	// rename below clears retErr on success, so we won't unlink the final
	// destination.
	defer func() {
		if retErr != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}
