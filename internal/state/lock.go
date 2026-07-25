package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/mermonia/peridot/internal/paths"
)

const LockTimeout = 5 * time.Second
const lockRetryInterval = 50 * time.Millisecond

func Acquire(dotfilesDir string) (func(), error) {
	lockPath := paths.LockFilePath(dotfilesDir)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("could not create parent dir: %w", err)
	}

	lock := flock.New(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), LockTimeout)
	defer cancel()

	locked, err := lock.TryLockContext(ctx, lockRetryInterval)
	if err != nil {
		return nil, fmt.Errorf("could not lock %s: %w", lockPath, err)
	}

	if !locked {
		return nil, fmt.Errorf("timed out after %s waiting for the state lock at %s, "+
			"another peridot process may be running", LockTimeout, lockPath)
	}

	released := false
	return func() {
		if released {
			return
		}
		released = true
		lock.Unlock()
	}, nil
}
