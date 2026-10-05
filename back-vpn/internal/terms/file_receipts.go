package terms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"darc-vpn/back-vpn/internal/domain"
)

type FileReceiptStore struct {
	directory string
}

func NewFileReceiptStore(directory string) (*FileReceiptStore, error) {
	if directory == "" || !filepath.IsAbs(directory) {
		return nil, errors.New("acceptance receipt directory must be an absolute configured path")
	}
	return &FileReceiptStore{directory: filepath.Clean(directory)}, nil
}

func (s *FileReceiptStore) Persist(ctx context.Context, receipt domain.TermsAcceptanceReceipt) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !safeID(receipt.EventID) {
		return errors.New("unsafe acceptance event ID")
	}
	if err := os.MkdirAll(s.directory, 0o700); err != nil {
		return fmt.Errorf("create receipt directory: %w", err)
	}
	if err := os.Chmod(s.directory, 0o700); err != nil {
		return fmt.Errorf("restrict receipt directory: %w", err)
	}
	temporary, err := os.CreateTemp(s.directory, ".tmp-"+receipt.EventID+"-")
	if err != nil {
		return fmt.Errorf("create temporary receipt: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if !committed {
			_ = temporary.Close()
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("restrict temporary receipt: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(receipt); err != nil {
		return fmt.Errorf("encode receipt: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary receipt: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary receipt: %w", err)
	}
	finalPath := filepath.Join(s.directory, receipt.EventID+".json")
	if _, err := os.Lstat(finalPath); err == nil {
		return errors.New("acceptance receipt already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect receipt target: %w", err)
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return fmt.Errorf("commit receipt: %w", err)
	}
	directory, err := os.Open(s.directory)
	if err != nil {
		return fmt.Errorf("open receipt directory for sync: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("sync receipt directory: %w", err)
	}
	if err := directory.Close(); err != nil {
		return fmt.Errorf("close receipt directory: %w", err)
	}
	committed = true
	return nil
}

func safeID(value string) bool {
	if value == "" || strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}
