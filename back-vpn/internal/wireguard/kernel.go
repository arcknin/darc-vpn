package wireguard

import (
	"context"
	"darc-vpn/back-vpn/internal/domain"
	"errors"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net"
	"regexp"
	"time"
)

type DeviceClient interface {
	Device(string) (*wgtypes.Device, error)
	ConfigureDevice(string, wgtypes.Config) error
	Close() error
}
type Kernel struct {
	name   string
	client DeviceClient
}

var interfaceName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,15}$`)

func NewKernel(name string) (*Kernel, error) {
	if !interfaceName.MatchString(name) {
		return nil, errors.New("invalid WireGuard interface")
	}
	client, err := wgctrl.New()
	if err != nil {
		return nil, errors.New("WireGuard control unavailable")
	}
	return NewKernelWithClient(name, client)
}
func NewKernelWithClient(name string, client DeviceClient) (*Kernel, error) {
	if !interfaceName.MatchString(name) || client == nil {
		return nil, errors.New("invalid WireGuard controller")
	}
	return &Kernel{name, client}, nil
}
func (k *Kernel) Close() error { return k.client.Close() }
func (k *Kernel) device(ctx context.Context) (*wgtypes.Device, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	d, err := k.client.Device(k.name)
	if err != nil {
		return nil, errors.New("WireGuard interface unavailable")
	}
	return d, nil
}
func (k *Kernel) ServerPublicKey(ctx context.Context) (string, error) {
	d, err := k.device(ctx)
	if err != nil {
		return "", err
	}
	if d.PublicKey == (wgtypes.Key{}) {
		return "", errors.New("WireGuard server identity unavailable")
	}
	return d.PublicKey.String(), nil
}
func (k *Kernel) Peers(ctx context.Context) (map[string][]net.IPNet, error) {
	d, err := k.device(ctx)
	if err != nil {
		return nil, err
	}
	result := map[string][]net.IPNet{}
	for _, p := range d.Peers {
		result[p.PublicKey.String()] = p.AllowedIPs
	}
	return result, nil
}
func (k *Kernel) EnsurePeer(ctx context.Context, key string, ips []net.IPNet) error {
	public, err := wgtypes.ParseKey(key)
	if err != nil {
		return errors.New("invalid peer identity")
	}
	if _, err = k.device(ctx); err != nil {
		return err
	}
	if err = k.client.ConfigureDevice(k.name, wgtypes.Config{Peers: []wgtypes.PeerConfig{{PublicKey: public, ReplaceAllowedIPs: true, AllowedIPs: ips}}}); err != nil {
		return errors.New("peer configuration failed")
	}
	peers, err := k.Peers(ctx)
	if err != nil {
		return err
	}
	actual, ok := peers[key]
	if !ok || !sameIPs(actual, ips) {
		return errors.New("peer configuration unconfirmed")
	}
	return nil
}
func sameIPs(a, b []net.IPNet) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]bool{}
	for _, ip := range a {
		m[ip.String()] = true
	}
	for _, ip := range b {
		if !m[ip.String()] {
			return false
		}
	}
	return true
}
func (k *Kernel) RemovePeers(ctx context.Context, keys []string) ([]domain.WireGuardRemoval, error) {
	if _, err := k.device(ctx); err != nil {
		return nil, err
	}
	var configs []wgtypes.PeerConfig
	for _, key := range keys {
		p, err := wgtypes.ParseKey(key)
		if err != nil {
			return nil, errors.New("invalid peer identity")
		}
		configs = append(configs, wgtypes.PeerConfig{PublicKey: p, Remove: true})
	}
	if err := k.client.ConfigureDevice(k.name, wgtypes.Config{Peers: configs}); err != nil {
		return nil, errors.New("peer removal failed")
	}
	peers, err := k.Peers(ctx)
	if err != nil {
		return nil, err
	}
	var result []domain.WireGuardRemoval
	for _, key := range keys {
		_, present := peers[key]
		result = append(result, domain.WireGuardRemoval{PeerID: key, Removed: !present})
	}
	return result, nil
}
func (k *Kernel) ConfirmPeersRemoved(ctx context.Context, keys []string) (bool, error) {
	peers, err := k.Peers(ctx)
	if err != nil {
		return false, err
	}
	for _, key := range keys {
		if _, ok := peers[key]; ok {
			return false, nil
		}
	}
	return true, nil
}

// EnsureLegacy explicitly restores PSK and keepalive, then verifies all fields.
func (k *Kernel) EnsureLegacy(ctx context.Context, key string, ips []net.IPNet, psk string, keepalive int) error {
	public, err := wgtypes.ParseKey(key)
	if err != nil {
		return errors.New("invalid legacy identity")
	}
	var shared wgtypes.Key
	if psk != "" {
		shared, err = wgtypes.ParseKey(psk)
		if err != nil {
			return errors.New("invalid legacy PSK")
		}
	}
	interval := time.Duration(keepalive) * time.Second
	before, err := k.device(ctx)
	if err != nil {
		return err
	}
	for _, p := range before.Peers {
		if p.PublicKey == public && p.PresharedKey == shared && p.PersistentKeepaliveInterval == interval && sameIPs(p.AllowedIPs, ips) {
			return nil
		}
	}
	if err = k.client.ConfigureDevice(k.name, wgtypes.Config{Peers: []wgtypes.PeerConfig{{PublicKey: public, PresharedKey: &shared, PersistentKeepaliveInterval: &interval, ReplaceAllowedIPs: true, AllowedIPs: ips}}}); err != nil {
		return errors.New("legacy configuration failed")
	}
	d, err := k.device(ctx)
	if err != nil {
		return err
	}
	for _, p := range d.Peers {
		if p.PublicKey == public && p.PresharedKey == shared && p.PersistentKeepaliveInterval == interval && sameIPs(p.AllowedIPs, ips) {
			return nil
		}
	}
	return errors.New("legacy configuration unconfirmed")
}
