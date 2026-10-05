// Operator-only offline import/reconciliation. Does not open an HTTP listener.
package main

import (
	"context"
	crypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/storage"
	"darc-vpn/back-vpn/internal/wireguard"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Legacy operation failed: input, lock, state or kernel validation rejected. No secrets printed.")
		os.Exit(1)
	}
}
func execute(args []string) error {
	f := flag.NewFlagSet("legacy-import", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	database := f.String("db", "", "SQLite path")
	master := f.String("master-key-file", "", "persistent key path")
	manifest := f.String("manifest", "", "private peer manifest")
	mode := f.String("operation", "import", "import or reconcile")
	iface := f.String("interface", "wg0", "isolated interface")
	pool4 := f.String("pool4", "10.66.66.0/24", "IPv4 pool")
	pool6 := f.String("pool6", "fd42:42:42::/64", "IPv6 pool")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 || !filepath.IsAbs(*database) || !filepath.IsAbs(*master) {
		return errors.New("absolute paths required")
	}
	if *mode != "import" && *mode != "reconcile" {
		return errors.New("invalid operation")
	}
	info, err := os.Stat(*master)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return errors.New("private master key file required")
	}
	raw, err := os.ReadFile(*master)
	if err != nil {
		return err
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return err
	}
	cipher, err := crypto.NewAESGCM(key, nil)
	if err != nil {
		return err
	}
	lock, err := wireguard.AcquireWriterLock(*database + ".writer.lock")
	if err != nil {
		return err
	}
	defer lock.Close()
	store, err := storage.Open(*database)
	if err != nil {
		return err
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err = store.Migrate(ctx); err != nil {
		return err
	}
	if *mode == "import" {
		info, err := os.Stat(*manifest)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 2<<20 {
			return errors.New("private manifest required")
		}
		file, err := os.Open(*manifest)
		if err != nil {
			return err
		}
		defer file.Close()
		var data struct {
			SchemaVersion int                 `json:"schemaVersion"`
			Peers         []domain.LegacyPeer `json:"peers"`
		}
		decoder := json.NewDecoder(file)
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&data); err != nil {
			return err
		}
		var extra any
		if err = decoder.Decode(&extra); err != io.EOF {
			return errors.New("extra manifest content")
		}
		if data.SchemaVersion != 1 {
			return errors.New("manifest version")
		}
		if err = store.ImportLegacy(ctx, data.Peers, cipher); err != nil {
			return err
		}
		fmt.Println("LEGACY_IMPORT_OK")
		return nil
	}
	if err = store.ConfigureManaged(ctx, *pool4, *pool6); err != nil {
		return err
	}
	kernel, err := wireguard.NewKernel(*iface)
	if err != nil {
		return err
	}
	defer kernel.Close()
	manager := wireguard.NewManager(store, kernel, 51442).WithLegacy(func(ctx context.Context) ([]domain.LegacyPeer, error) { return store.LegacyPeers(ctx, cipher) })
	if err = manager.Sync(ctx); err != nil {
		return err
	}
	if !manager.Ready() {
		return errors.New("not ready")
	}
	fmt.Println("LEGACY_RECONCILE_OK")
	return nil
}
