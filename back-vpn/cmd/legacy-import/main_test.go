package main

import (
	"bytes"
	"darc-vpn/back-vpn/internal/domain"
	"encoding/hex"
	"encoding/json"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"os"
	"path/filepath"
	"testing"
)

func TestOfflineImportIsPrivateAndIdempotent(t *testing.T) {
	root := t.TempDir()
	master := filepath.Join(root, "master.hex")
	manifest := filepath.Join(root, "peers.json")
	db := filepath.Join(root, "vpn.db")
	os.WriteFile(master, []byte(hex.EncodeToString(bytes.Repeat([]byte{7}, 32))), 0600)
	key, _ := wgtypes.GeneratePrivateKey()
	psk, _ := wgtypes.GenerateKey()
	data, _ := json.Marshal(struct {
		SchemaVersion int                 `json:"schemaVersion"`
		Peers         []domain.LegacyPeer `json:"peers"`
	}{1, []domain.LegacyPeer{{PublicKey: key.PublicKey().String(), AllowedIPs: []string{"10.66.66.2/32"}, PresharedKey: psk.String()}}})
	os.WriteFile(manifest, data, 0600)
	args := []string{"--db", db, "--master-key-file", master, "--manifest", manifest}
	if err := execute(args); err != nil {
		t.Fatal(err)
	}
	if err := execute(args); err != nil {
		t.Fatal(err)
	}
	os.Chmod(manifest, 0644)
	if err := execute(args); err == nil {
		t.Fatal("public manifest accepted")
	}
}
