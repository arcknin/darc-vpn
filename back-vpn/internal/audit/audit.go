package audit

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"

	"darc-vpn/back-vpn/internal/domain"
)

var ErrControllerUnauthenticated = errors.New("VPN controller is not authenticated")

type Event struct {
	Type      string
	AccountID string
	ProfileID string
	SourceIP  string
}

type MetadataSink interface {
	Record(context.Context, domain.AccessMetadata) error
}

type Recorder struct {
	sink  MetadataSink
	clock domain.Clock
	ids   domain.IDGenerator
}

func NewRecorder(sink MetadataSink, clock domain.Clock, ids domain.IDGenerator) *Recorder {
	return &Recorder{sink: sink, clock: clock, ids: ids}
}

func (r *Recorder) Record(ctx context.Context, event Event) error {
	if r == nil || r.sink == nil || r.clock == nil || r.ids == nil || strings.TrimSpace(event.Type) == "" {
		return errors.New("audit recorder is not configured")
	}
	eventID, err := r.ids.NewID()
	if err != nil {
		return err
	}
	return r.sink.Record(ctx, domain.AccessMetadata{
		EventID: eventID, EventType: event.Type, AccountID: event.AccountID,
		ProfileID: event.ProfileID, SourceIP: event.SourceIP, At: r.clock.Now().UTC(),
	})
}

type JSONSink struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewJSONSink(writer io.Writer) *JSONSink { return &JSONSink{writer: writer} }

type jsonMetadata struct {
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	AccountID    string `json:"account_id,omitempty"`
	ProfileID    string `json:"profile_id,omitempty"`
	SourceIP     string `json:"source_ip"`
	TimestampUTC string `json:"timestamp_utc"`
}

func (s *JSONSink) Record(_ context.Context, metadata domain.AccessMetadata) error {
	if s == nil || s.writer == nil {
		return errors.New("audit sink is not configured")
	}
	value := jsonMetadata{
		EventID: metadata.EventID, EventType: metadata.EventType, AccountID: metadata.AccountID,
		ProfileID: metadata.ProfileID, SourceIP: metadata.SourceIP, TimestampUTC: metadata.At.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.NewEncoder(s.writer).Encode(value)
}

type SourceIPResolver struct {
	trusted []netip.Prefix
}

func NewSourceIPResolver(trustedProxies []netip.Prefix) *SourceIPResolver {
	return &SourceIPResolver{trusted: append([]netip.Prefix(nil), trustedProxies...)}
}

func (r *SourceIPResolver) Resolve(request *http.Request) string {
	remote := parseAddress(request.RemoteAddr)
	if !remote.IsValid() {
		return ""
	}
	trusted := false
	for _, prefix := range r.trusted {
		if prefix.Contains(remote) {
			trusted = true
			break
		}
	}
	if trusted {
		forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0])
		if address, err := netip.ParseAddr(forwarded); err == nil {
			return address.Unmap().String()
		}
	}
	return remote.Unmap().String()
}

func parseAddress(value string) netip.Addr {
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	address, _ := netip.ParseAddr(strings.Trim(value, "[]"))
	return address
}

type VPNEventPort struct {
	credential string
	recorder   *Recorder
}

func NewVPNEventPort(credential string, recorder *Recorder) (*VPNEventPort, error) {
	if credential == "" || recorder == nil {
		return nil, errors.New("VPN event port configuration is incomplete")
	}
	return &VPNEventPort{credential: credential, recorder: recorder}, nil
}

func (p *VPNEventPort) Record(ctx context.Context, credential, accountID, profileID, sourceIP string) error {
	if len(credential) != len(p.credential) || subtle.ConstantTimeCompare([]byte(credential), []byte(p.credential)) != 1 {
		return ErrControllerUnauthenticated
	}
	return p.recorder.Record(ctx, Event{Type: "vpn_event", AccountID: accountID, ProfileID: profileID, SourceIP: sourceIP})
}
