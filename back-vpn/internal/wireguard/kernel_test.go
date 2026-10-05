package wireguard

import (
	"context"
	"errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net"
	"testing"
	"time"
)

type deviceStub struct {
	device *wgtypes.Device
	fail   bool
	lie    bool
}

func (s *deviceStub) Device(string) (*wgtypes.Device, error) {
	if s.fail {
		return nil, errors.New("PRIVATE_SECRET")
	}
	return s.device, nil
}
func (s *deviceStub) ConfigureDevice(_ string, c wgtypes.Config) error {
	if s.fail {
		return errors.New("PRIVATE_SECRET")
	}
	if s.lie {
		return nil
	}
	for _, p := range c.Peers {
		if p.Remove {
			var left []wgtypes.Peer
			for _, old := range s.device.Peers {
				if old.PublicKey != p.PublicKey {
					left = append(left, old)
				}
			}
			s.device.Peers = left
		} else {
			var others []wgtypes.Peer
			for _, old := range s.device.Peers {
				if old.PublicKey != p.PublicKey {
					others = append(others, old)
				}
			}
			s.device.Peers = append([]wgtypes.Peer{{PublicKey: p.PublicKey, AllowedIPs: p.AllowedIPs}}, others...)
			if p.PresharedKey != nil {
				s.device.Peers[0].PresharedKey = *p.PresharedKey
			}
			if p.PersistentKeepaliveInterval != nil {
				s.device.Peers[0].PersistentKeepaliveInterval = *p.PersistentKeepaliveInterval
			}
		}
	}
	return nil
}
func (s *deviceStub) Close() error { return nil }
func TestKernelProvisionAndRemovalRequireObservedState(t *testing.T) {
	key, _ := wgtypes.GeneratePrivateKey()
	server, _ := wgtypes.GeneratePrivateKey()
	stub := &deviceStub{device: &wgtypes.Device{PublicKey: server.PublicKey()}}
	k, err := NewKernelWithClient("wg0", stub)
	if err != nil {
		t.Fatal(err)
	}
	_, cidr, _ := net.ParseCIDR("10.77.77.2/32")
	if err = k.EnsurePeer(context.Background(), key.PublicKey().String(), []net.IPNet{*cidr}); err != nil {
		t.Fatal(err)
	}
	stub.lie = true
	key2, _ := wgtypes.GeneratePrivateKey()
	if err = k.EnsurePeer(context.Background(), key2.PublicKey().String(), []net.IPNet{*cidr}); err == nil {
		t.Fatal("unobserved install accepted")
	}
	result, err := k.RemovePeers(context.Background(), []string{key.PublicKey().String()})
	if err != nil || result[0].Removed {
		t.Fatal("unobserved removal accepted")
	}
	stub.lie = false
	result, err = k.RemovePeers(context.Background(), []string{key.PublicKey().String()})
	if err != nil || !result[0].Removed {
		t.Fatal("real removal not confirmed")
	}
	ok, err := k.ConfirmPeersRemoved(context.Background(), []string{key.PublicKey().String()})
	if err != nil || !ok {
		t.Fatal("absence not confirmed")
	}
	stub.fail = true
	if _, err = k.ServerPublicKey(context.Background()); err == nil || err.Error() == "PRIVATE_SECRET" {
		t.Fatal("unsanitized/missing failure")
	}
}
func TestWriterExclusivity(t *testing.T) {
	path := t.TempDir() + "/writer.lock"
	first, err := AcquireWriterLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := AcquireWriterLock(path); err == nil {
		second.Close()
		t.Fatal("second writer accepted")
	}
	first.Close()
	third, err := AcquireWriterLock(path)
	if err != nil {
		t.Fatal("lock did not release")
	}
	third.Close()
}
func TestKernelRestoresLegacyPSKAndKeepalive(t *testing.T) {
	key, _ := wgtypes.GeneratePrivateKey()
	server, _ := wgtypes.GeneratePrivateKey()
	psk, _ := wgtypes.GenerateKey()
	stub := &deviceStub{device: &wgtypes.Device{PublicKey: server.PublicKey()}}
	kernel, _ := NewKernelWithClient("wg0", stub)
	_, cidr, _ := net.ParseCIDR("10.66.66.2/32")
	if err := kernel.EnsureLegacy(context.Background(), key.PublicKey().String(), []net.IPNet{*cidr}, psk.String(), 25); err != nil {
		t.Fatal(err)
	}
	if stub.device.Peers[0].PresharedKey != psk || stub.device.Peers[0].PersistentKeepaliveInterval != 25*time.Second {
		t.Fatal("PSK or keepalive lost")
	}
}
