package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/audit"
	"darc-vpn/back-vpn/internal/auth"
	profilecrypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	httpapi "darc-vpn/back-vpn/internal/http"
	"darc-vpn/back-vpn/internal/profiles"
	"darc-vpn/back-vpn/internal/revocation"
	"darc-vpn/back-vpn/internal/storage"
	"darc-vpn/back-vpn/internal/terms"
	"darc-vpn/back-vpn/internal/wireguard"
)

type config struct {
	kernelClient         wireguard.DeviceClient
	retryInitial         time.Duration
	retryMax             time.Duration
	mode                 string
	wireGuardMode        string
	interfaceName        string
	pool4                string
	pool6                string
	endpointPort         int
	address              string
	databasePath         string
	masterKey            []byte
	termsVersion         string
	termsPath            string
	termsSHA256          string
	receiptDirectory     string
	adminUsername        string
	adminPassword        string
	controllerCredential string
	trustedProxies       []netip.Prefix
	passwordParams       auth.Argon2idParams
	wireGuardController  domain.WireGuardController
}

type application struct {
	writer    *wireguard.WriterLock
	kernel    *wireguard.Kernel
	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
	handler   http.Handler
	store     *storage.Store
	vpnEvents *audit.VPNEventPort
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "health" {
		response, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://127.0.0.1:8080/readyz")
		if err != nil {
			os.Exit(1)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}

	configuration, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	app, err := newApplication(context.Background(), configuration, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	server := &http.Server{Addr: configuration.address, Handler: app.handler, ReadHeaderTimeout: 5 * time.Second}
	log.Print("vpn API started")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadConfig(getenv func(string) string) (config, error) {
	configuration := config{
		mode: valueOr(getenv("VPN_MODE"), "local"), wireGuardMode: strings.TrimSpace(getenv("VPN_WIREGUARD_MODE")), interfaceName: valueOr(getenv("VPN_WG_INTERFACE"), "wg0"), pool4: getenv("VPN_POOL4"), pool6: getenv("VPN_POOL6"),
		address:              valueOr(getenv("API_ADDR"), "127.0.0.1:8080"),
		databasePath:         strings.TrimSpace(getenv("VPN_DB_PATH")),
		termsVersion:         strings.TrimSpace(getenv("VPN_TERMS_VERSION")),
		termsPath:            strings.TrimSpace(getenv("VPN_TERMS_PATH")),
		termsSHA256:          strings.TrimSpace(getenv("VPN_TERMS_SHA256")),
		receiptDirectory:     strings.TrimSpace(getenv("VPN_ACCEPTANCE_DIR")),
		adminUsername:        strings.TrimSpace(getenv("VPN_ADMIN_USERNAME")),
		adminPassword:        getenv("VPN_ADMIN_PASSWORD"),
		controllerCredential: getenv("VPN_CONTROLLER_AUDIT_TOKEN"),
		passwordParams:       auth.DefaultParams(),
	}
	port, err := strconv.Atoi(valueOr(getenv("VPN_ENDPOINT_PORT"), "51820"))
	if err != nil {
		return config{}, errors.New("invalid endpoint port")
	}
	configuration.endpointPort = port
	initial, err := time.ParseDuration(valueOr(getenv("VPN_RETRY_INITIAL"), "5s"))
	if err != nil || initial < 100*time.Millisecond {
		return config{}, errors.New("invalid reconciliation retry interval")
	}
	maximum, err := time.ParseDuration(valueOr(getenv("VPN_RETRY_MAX"), "60s"))
	if err != nil || maximum < initial || maximum > 5*time.Minute {
		return config{}, errors.New("invalid reconciliation retry maximum")
	}
	configuration.retryInitial = initial
	configuration.retryMax = maximum
	key, err := hex.DecodeString(strings.TrimSpace(getenv("VPN_MASTER_KEY_HEX")))
	if err != nil || len(key) != 32 {
		return config{}, errors.New("VPN_MASTER_KEY_HEX must contain exactly 32 bytes as hexadecimal")
	}
	configuration.masterKey = key
	for _, item := range strings.Split(getenv("VPN_TRUSTED_PROXY_CIDRS"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			return config{}, fmt.Errorf("invalid trusted proxy CIDR: %w", err)
		}
		configuration.trustedProxies = append(configuration.trustedProxies, prefix)
	}
	if err := validateConfig(configuration); err != nil {
		return config{}, err
	}
	return configuration, nil
}

func validateConfig(configuration config) error {
	if configuration.mode != "" && configuration.mode != "local" && configuration.mode != "container-lab" && configuration.mode != "container-private" && configuration.mode != "container-private-test" {
		return errors.New("invalid execution mode")
	}
	if configuration.wireGuardMode != "fake" && configuration.wireGuardMode != "kernel" {
		return errors.New("explicit WireGuard mode required")
	}
	if (configuration.mode == "container-lab" || configuration.mode == "container-private" || configuration.mode == "container-private-test") && configuration.wireGuardMode != "kernel" {
		return errors.New("container lab requires kernel WireGuard")
	}
	if configuration.wireGuardMode == "kernel" && (configuration.interfaceName != "wg0" || configuration.pool4 == "" || configuration.pool6 == "" || configuration.endpointPort < 1 || configuration.endpointPort > 65535) {
		return errors.New("kernel WireGuard configuration incomplete")
	}
	if configuration.databasePath == "" || !filepath.IsAbs(configuration.databasePath) {
		return errors.New("VPN_DB_PATH must be an absolute configured path")
	}
	if configuration.mode != "container-private-test" && (configuration.termsVersion == "" || configuration.termsPath == "" || !filepath.IsAbs(configuration.termsPath) || configuration.termsSHA256 == "") {
		return errors.New("terms version, absolute document path, and SHA-256 are required")
	}
	if configuration.receiptDirectory == "" || !filepath.IsAbs(configuration.receiptDirectory) {
		return errors.New("VPN_ACCEPTANCE_DIR must be an absolute configured path")
	}
	if configuration.adminUsername == "" || configuration.adminPassword == "" {
		return errors.New("initial administrator credentials are required")
	}
	if configuration.controllerCredential == "" {
		return errors.New("VPN controller audit credential is required")
	}
	if len(configuration.masterKey) != 32 {
		return errors.New("profile master key must contain exactly 32 bytes")
	}
	host, _, err := net.SplitHostPort(configuration.address)
	if err != nil {
		return errors.New("API_ADDR must include a loopback host and port")
	}
	address, err := netip.ParseAddr(host)
	if err != nil || (!address.IsLoopback() && !((configuration.mode == "container-lab" || configuration.mode == "container-private" || configuration.mode == "container-private-test") && host == "0.0.0.0")) {
		return errors.New("the local unencrypted API may bind only to loopback")
	}
	if executable, err := os.Executable(); err == nil && within(configuration.receiptDirectory, filepath.Dir(executable)) {
		return errors.New("acceptance receipts must be configured outside the executable build directory")
	}
	return nil
}

func newApplication(ctx context.Context, configuration config, auditOutput io.Writer) (*application, error) {
	if err := validateConfig(configuration); err != nil {
		return nil, err
	}
	var document domain.TermsDocument
	var err error
	if configuration.mode == "container-private-test" {
		contents := []byte("Teste técnico privado do Darc VPN, restrito ao operador. Termos finais e retenção não configurados; não liberar para usuários. Este aviso não representa aceite dos termos definitivos.")
		sum := sha256.Sum256(contents)
		document, err = terms.NewDocument("private-technical-test", contents, hex.EncodeToString(sum[:]))
	} else {
		contents, readErr := os.ReadFile(configuration.termsPath)
		if readErr != nil {
			return nil, fmt.Errorf("read configured terms: %w", readErr)
		}
		document, err = terms.NewDocument(configuration.termsVersion, contents, configuration.termsSHA256)
	}
	if err != nil {
		return nil, err
	}
	receipts, err := terms.NewFileReceiptStore(configuration.receiptDirectory)
	if err != nil {
		return nil, err
	}
	var writer *wireguard.WriterLock
	var kernel *wireguard.Kernel
	if configuration.wireGuardMode == "kernel" {
		writer, err = wireguard.AcquireWriterLock(configuration.databasePath + ".writer.lock")
		if err != nil {
			return nil, err
		}
	}
	store, err := storage.Open(configuration.databasePath)
	if err != nil {
		if writer != nil {
			writer.Close()
		}
		return nil, err
	}
	fail := func(err error) (*application, error) {
		if kernel != nil {
			kernel.Close()
		}
		if writer != nil {
			writer.Close()
		}
		_ = store.Close()
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		return fail(err)
	}
	if configuration.wireGuardMode == "kernel" {
		if err = store.ConfigureManaged(ctx, configuration.pool4, configuration.pool6); err != nil {
			return fail(err)
		}
	}
	cipher, err := profilecrypto.NewAESGCM(configuration.masterKey, nil)
	if err != nil {
		return fail(err)
	}
	clock := realClock{}
	ids := profilecrypto.NewOpaqueIDGenerator(nil)
	hasher := auth.NewArgon2idHasher(configuration.passwordParams)
	factory := profiles.NewFactory(profiles.FactoryConfig{
		IDs: ids, Clock: clock, Keys: profiles.NewWireGuardKeyGenerator(nil), Cipher: cipher,
		Address: addressForProfile, AllowedIPs: []string{"0.0.0.0/0", "::/0"}, DNS: []string{"1.1.1.1", "2606:4700:4700::1111"},
	})
	accountService := accounts.NewService(store, hasher, factory, ids, clock)
	adminExists, err := store.AdminExists(ctx, configuration.adminUsername)
	if err != nil {
		return fail(fmt.Errorf("check initial administrator: %w", err))
	}
	if !adminExists {
		if _, err := accountService.BootstrapAdmin(ctx, configuration.adminUsername, configuration.adminPassword); err != nil {
			return fail(fmt.Errorf("bootstrap administrator: %w", err))
		}
	}
	authService, err := auth.NewService(store, hasher, clock, auth.Config{SessionTTL: 24 * time.Hour})
	if err != nil {
		return fail(err)
	}
	termsService, err := terms.NewService(document, store, receipts, clock, ids)
	if err != nil {
		return fail(err)
	}
	if configuration.mode == "container-private-test" {
		termsService.WithAcceptanceRequired(false)
	}
	var deliveryGate profiles.AcceptanceGate = termsService
	if configuration.mode == "container-private-test" {
		deliveryGate = privateTestDeliveryGate{}
	}
	profileService := profiles.NewService(store, cipher, deliveryGate)
	var manager *wireguard.Manager
	controller := configuration.wireGuardController
	if configuration.wireGuardMode == "kernel" {
		if configuration.kernelClient != nil {
			kernel, err = wireguard.NewKernelWithClient(configuration.interfaceName, configuration.kernelClient)
		} else {
			kernel, err = wireguard.NewKernel(configuration.interfaceName)
		}
		if err != nil {
			return fail(err)
		}
		manager = wireguard.NewManager(store, kernel, configuration.endpointPort).WithLegacy(func(ctx context.Context) ([]domain.LegacyPeer, error) { return store.LegacyPeers(ctx, cipher) })
		controller = manager
		accountService.WithProvisioning(func(ctx context.Context) { _ = manager.Sync(ctx) })
		profileService.WithDeliveryMetadata(manager.DeliveryMetadata)
	}
	if controller == nil {
		controller = wireguard.NewFake(wireguard.FakeConfirmAll)
	}
	revocationService := revocation.NewService(store, controller, clock)
	recorder := audit.NewRecorder(audit.NewJSONSink(auditOutput), clock, ids)
	vpnEvents, err := audit.NewVPNEventPort(configuration.controllerCredential, recorder)
	if err != nil {
		return fail(err)
	}
	handler := httpapi.New(authService, accountService, profileService,
		httpapi.WithTerms(termsService),
		httpapi.WithRevocation(revocationService),
		httpapi.WithAudit(recorder, audit.NewSourceIPResolver(configuration.trustedProxies)),
	)
	app := &application{handler: handler, store: store, vpnEvents: vpnEvents, writer: writer, kernel: kernel}
	if manager != nil {
		syncPeers := func(ctx context.Context) error {
			if err := manager.Sync(ctx); err != nil {
				return err
			}
			blocked, err := store.BlockedAccounts(ctx)
			if err != nil {
				return err
			}
			for _, id := range blocked {
				if _, err = revocationService.Block(ctx, auth.Principal{Role: domain.RoleAdmin}, id); err != nil {
					return err
				}
			}
			return nil
		}
		// Serialize HTTP access, reconciliation and revocation around external effects.
		app.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				handler.ServeHTTP(w, r)
				return
			}
			manager.Mu.Lock()
			defer manager.Mu.Unlock()
			if r.URL.Path == "/readyz" {
				if err := syncPeers(r.Context()); err != nil || !manager.Ready() {
					http.Error(w, "not ready", http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			handler.ServeHTTP(w, r)
		})
		manager.Mu.Lock()
		_ = syncPeers(ctx)
		manager.Mu.Unlock()
		loopCtx, cancel := context.WithCancel(context.Background())
		app.cancel = cancel
		app.done = make(chan struct{})
		go func() {
			defer close(app.done)
			initial := configuration.retryInitial
			if initial == 0 {
				initial = 5 * time.Second
			}
			maximum := configuration.retryMax
			if maximum == 0 {
				maximum = 60 * time.Second
			}
			delay := initial
			for {
				timer := time.NewTimer(delay)
				select {
				case <-loopCtx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				manager.Mu.Lock()
				err := syncPeers(loopCtx)
				manager.Mu.Unlock()
				if err != nil {
					delay *= 2
					if delay > maximum {
						delay = maximum
					}
				} else {
					delay = initial
				}
			}
		}()
	} else {
		app.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/readyz" {
				w.WriteHeader(http.StatusOK)
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
	return app, nil
}

func (a *application) Close() error {
	if a == nil || a.store == nil {
		return nil
	}
	var err error
	a.closeOnce.Do(func() {
		if a.cancel != nil {
			a.cancel()
			<-a.done
		}
		if a.kernel != nil {
			a.kernel.Close()
		}
		err = a.store.Close()
		if a.writer != nil {
			a.writer.Close()
		}
	})
	return err
}

func addressForProfile(_, profileID string) string {
	digest := sha256.Sum256([]byte(profileID))
	host := 2 + int(digest[0])%253
	return fmt.Sprintf("10.8.0.%d/32", host)
}

func within(path, parent string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// Private-test bypass affects acceptance only; authentication, ownership and kernel readiness remain enforced.
type privateTestDeliveryGate struct{}

func (privateTestDeliveryGate) AuthorizeProfileDelivery(context.Context, string) error { return nil }
