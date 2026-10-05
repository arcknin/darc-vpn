package main

import (
	"context"
	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/terms"
	"encoding/json"
	"errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"io"
	"net/http"
	"sync"
	"testing"
)

type stackDevice struct {
	mu   sync.Mutex
	d    wgtypes.Device
	fail bool
}

func (s *stackDevice) Device(string) (*wgtypes.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return nil, errors.New("secret error")
	}
	copy := s.d
	copy.Peers = append([]wgtypes.Peer(nil), s.d.Peers...)
	return &copy, nil
}
func (s *stackDevice) ConfigureDevice(_ string, c wgtypes.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("secret error")
	}
	for _, p := range c.Peers {
		var kept []wgtypes.Peer
		for _, old := range s.d.Peers {
			if old.PublicKey != p.PublicKey {
				kept = append(kept, old)
			}
		}
		if !p.Remove {
			kept = append(kept, wgtypes.Peer{PublicKey: p.PublicKey, AllowedIPs: p.AllowedIPs})
		}
		s.d.Peers = kept
	}
	return nil
}
func (s *stackDevice) Close() error { return nil }
func TestStackHTTPProvisioningRevocationAndRestart(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	server, _ := wgtypes.GeneratePrivateKey()
	device := &stackDevice{d: wgtypes.Device{PublicKey: server.PublicKey()}}
	cfg.wireGuardMode = "kernel"
	cfg.interfaceName = "wg0"
	cfg.pool4 = "10.77.77.0/24"
	cfg.pool6 = "fd42:77:77::/64"
	cfg.endpointPort = 51443
	cfg.kernelClient = device
	app, err := newApplication(ctx, cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { app.Close() }()
	if other, err := newApplication(ctx, cfg, io.Discard); err == nil {
		other.Close()
		t.Fatal("second writer started")
	}
	admin := loginOnHandler(t, app.handler, "/api/v1/auth/admin/login", "root", "admin-pass")
	response := serve(t, app.handler, http.MethodPost, "/api/v1/admin/accounts", admin, accounts.CreateRequest{Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Primary"})
	if response.Code != 201 {
		t.Fatalf("create %d", response.Code)
	}
	var account domain.AdminAccountMetadata
	json.Unmarshal(response.Body.Bytes(), &account)
	if account.Profiles[0].ProvisioningState != "ready" {
		t.Fatal("peer not ready")
	}
	friend := loginOnHandler(t, app.handler, "/api/v1/auth/friend/login", "friend", "friend-pass")
	path := "/api/v1/profiles/" + account.PrimaryID + "/private"
	if got := serve(t, app.handler, "GET", path, friend, nil); got.Code != 428 {
		t.Fatal("secret before terms")
	}
	docResp := serve(t, app.handler, "GET", "/api/v1/terms/current", friend, nil)
	var doc domain.TermsDocument
	json.Unmarshal(docResp.Body.Bytes(), &doc)
	serve(t, app.handler, "POST", "/api/v1/terms/acceptance", friend, terms.AcceptanceRequest{Accepted: true, Version: doc.Version, SHA256: doc.SHA256})
	priv := serve(t, app.handler, "GET", path, friend, nil)
	var p domain.ProfilePrivate
	json.Unmarshal(priv.Body.Bytes(), &p)
	if priv.Code != 200 || p.ServerPublicKey != server.PublicKey().String() || p.EndpointPort != 51443 || p.ServerPublicKey == p.PeerPublicKey {
		t.Fatal("incorrect server metadata")
	}
	device.mu.Lock()
	device.fail = true
	device.mu.Unlock()
	if got := serve(t, app.handler, "GET", path, friend, nil); got.Code != 503 {
		t.Fatal("secret delivered with missing kernel peer")
	}
	if got := serve(t, app.handler, "GET", "/readyz", "", nil); got.Code != 503 {
		t.Fatal("false readiness")
	}
	device.mu.Lock()
	device.fail = false
	device.mu.Unlock()
	if got := serve(t, app.handler, "POST", "/api/v1/admin/accounts/"+account.ID+"/block", admin, nil); got.Code != 202 {
		t.Fatal("block failed")
	}
	if got := serve(t, app.handler, "GET", path, friend, nil); got.Code != 401 {
		t.Fatal("session not cut")
	}
	app.Close()
	device.mu.Lock()
	device.d.Peers = nil
	device.mu.Unlock()
	app, err = newApplication(ctx, cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if got := serve(t, app.handler, "GET", "/readyz", "", nil); got.Code != 200 {
		t.Fatal("restart not ready")
	}
	d, _ := device.Device("wg0")
	if len(d.Peers) != 0 {
		t.Fatal("revoked peers resurrected")
	}
}

func TestStackConcurrentProvisionReconcileAndBlock(t *testing.T) {
	cfg := testConfig(t)
	server, _ := wgtypes.GeneratePrivateKey()
	device := &stackDevice{d: wgtypes.Device{PublicKey: server.PublicKey()}}
	cfg.wireGuardMode = "kernel"
	cfg.interfaceName = "wg0"
	cfg.pool4 = "10.77.77.0/24"
	cfg.pool6 = "fd42:77:77::/64"
	cfg.endpointPort = 51443
	cfg.kernelClient = device
	app, err := newApplication(context.Background(), cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	admin := loginOnHandler(t, app.handler, "/api/v1/auth/admin/login", "root", "admin-pass")
	created := serve(t, app.handler, "POST", "/api/v1/admin/accounts", admin, accounts.CreateRequest{Username: "race", Password: "fixture-password", PrimaryDisplayName: "Primary"})
	var account domain.AdminAccountMetadata
	if err = json.Unmarshal(created.Body.Bytes(), &account); err != nil || created.Code != 201 {
		t.Fatal("account setup failed")
	}
	var wg sync.WaitGroup
	failures := make(chan string, 21)
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			r := serve(t, app.handler, "POST", "/api/v1/admin/accounts/"+account.ID+"/profiles", admin, map[string]string{"displayName": "Extra"})
			if r.Code != 201 && r.Code != 423 {
				failures <- "unexpected create status"
			}
		}()
		go func() { defer wg.Done(); serve(t, app.handler, "GET", "/readyz", "", nil) }()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := serve(t, app.handler, "POST", "/api/v1/admin/accounts/"+account.ID+"/block", admin, nil)
		if r.Code != 202 {
			failures <- "block failed"
		}
	}()
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	serve(t, app.handler, "GET", "/readyz", "", nil)
	d, err := device.Device("wg0")
	if err != nil || len(d.Peers) != 0 {
		t.Fatal("race resurrected blocked peer")
	}
	peers, err := app.store.ManagedPeers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range peers {
		if p.State != "revoked" || p.Active {
			t.Fatal("race lost durable revocation")
		}
	}
}

func TestPrivateTestStartsAndRecoversProfileWithoutTermsOrAcceptance(t *testing.T) {
	cfg := testConfig(t)
	cfg.mode = "container-private-test"
	cfg.wireGuardMode = "kernel"
	cfg.interfaceName = "wg0"
	cfg.pool4 = "10.77.77.0/24"
	cfg.pool6 = "fd42:77:77::/64"
	cfg.endpointPort = 51443
	cfg.termsPath = ""
	cfg.termsVersion = ""
	cfg.termsSHA256 = ""
	server, _ := wgtypes.GeneratePrivateKey()
	cfg.kernelClient = &stackDevice{d: wgtypes.Device{PublicKey: server.PublicKey()}}
	app, err := newApplication(context.Background(), cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	admin := loginOnHandler(t, app.handler, "/api/v1/auth/admin/login", "root", "admin-pass")
	r := serve(t, app.handler, "POST", "/api/v1/admin/accounts", admin, accounts.CreateRequest{Username: "private-tester", Password: "fixture-password", PrimaryDisplayName: "Primary"})
	var account domain.AdminAccountMetadata
	json.Unmarshal(r.Body.Bytes(), &account)
	if r.Code != 201 {
		t.Fatal("account create failed")
	}
	friend := loginOnHandler(t, app.handler, "/api/v1/auth/friend/login", "private-tester", "fixture-password")
	if r := serve(t, app.handler, "GET", "/api/v1/profiles/"+account.PrimaryID+"/private", friend, nil); r.Code != 200 {
		t.Fatal("terms still block private test")
	}
	cfg.mode = "container-private"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("normal mode lost terms requirements")
	}
}
