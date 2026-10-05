package wireguard

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

type WriterLock struct{ file *os.File }

func AcquireWriterLock(path string) (*WriterLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, errors.New("writer lock unavailable")
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("another peer writer is active")
	}
	return &WriterLock{f}, nil
}
func (l *WriterLock) Close() error {
	if l == nil {
		return nil
	}
	return l.file.Close()
}
