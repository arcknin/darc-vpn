package storage

import (
	"bytes"
	"context"
	crypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"path/filepath"
	"testing"
)

func TestLegacyImportEncryptedIdempotentConflictAtomicAndReservesIPs(t *testing.T) {
	ctx := context.Background()
	s := managedStore(t, "10.77.77.0/29")
	cipher, _ := crypto.NewAESGCM(bytes.Repeat([]byte{7}, 32), nil)
	psk, _ := wgtypes.GenerateKey()
	key, _ := wgtypes.GeneratePrivateKey()
	peers := []domain.LegacyPeer{{PublicKey: key.PublicKey().String(), AllowedIPs: []string{"10.77.77.2/32"}, PresharedKey: psk.String(), Keepalive: 25}}
	if err := s.ImportLegacy(ctx, peers, cipher); err != nil {
		t.Fatal(err)
	}
	if err := s.ImportLegacy(ctx, peers, cipher); err != nil {
		t.Fatal(err)
	}
	actual, err := s.LegacyPeers(ctx, cipher)
	if err != nil || len(actual) != 1 || actual[0].PresharedKey != psk.String() {
		t.Fatal("legacy not restored")
	}
	var data []byte
	if err = s.DB().QueryRow("SELECT psk_ciphertext FROM legacy_peers").Scan(&data); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(psk.String())) {
		t.Fatal("PSK plaintext in database")
	}
	k2, _ := wgtypes.GeneratePrivateKey()
	changed := append([]domain.LegacyPeer{{PublicKey: k2.PublicKey().String(), AllowedIPs: []string{"10.77.77.3/32"}}}, peers...)
	changed[1].Keepalive = 30
	if err := s.ImportLegacy(ctx, changed, cipher); err == nil {
		t.Fatal("conflicting update accepted")
	}
	actual, _ = s.LegacyPeers(ctx, cipher)
	if len(actual) != 1 {
		t.Fatal("partial import persisted")
	}
	if err := createManaged(s, 0); err != nil {
		t.Fatal(err)
	}
	profile, _ := s.Profile(ctx, "0", "0")
	if profile.Address == "10.77.77.2/32, fd42:77:77::2/128" {
		t.Fatal("allocated legacy address")
	}
	wrong, _ := crypto.NewAESGCM(bytes.Repeat([]byte{9}, 32), nil)
	if _, err = s.LegacyPeers(ctx, wrong); err == nil {
		t.Fatal("wrong key accepted")
	}

}
func TestLegacyImportAfterRestartAndManagedOverlapRejected(t *testing.T) {
	ctx := context.Background()
	file := filepath.Join(t.TempDir(), "vpn.db")
	s, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	s.Migrate(ctx)
	s.ConfigureManaged(ctx, "10.77.77.0/24", "fd42:77:77::/64")
	cipher, _ := crypto.NewAESGCM(bytes.Repeat([]byte{8}, 32), nil)
	if err = createManaged(s, 0); err != nil {
		t.Fatal(err)
	}
	key, _ := wgtypes.GeneratePrivateKey()
	if err = s.ImportLegacy(ctx, []domain.LegacyPeer{{PublicKey: key.PublicKey().String(), AllowedIPs: []string{"10.77.77.0/24"}}}, cipher); err == nil {
		t.Fatal("managed address collision accepted")
	}
	if err = s.ImportLegacy(ctx, []domain.LegacyPeer{{PublicKey: key.PublicKey().String(), AllowedIPs: []string{"fd42:77:77::42/128"}}}, cipher); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	actual, err := s.LegacyPeers(ctx, cipher)
	if err != nil || len(actual) != 1 {
		t.Fatal("legacy disappeared")
	}
}
