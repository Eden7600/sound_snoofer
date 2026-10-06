// Package hue controls Philips Hue scenes and room lighting through a local
// Hue Bridge using the CLIP v2 API.
package hue

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 5 * time.Second

var (
	// ErrCertificateChanged reports a bridge certificate that differs from the paired pin.
	ErrCertificateChanged = errors.New("bridge certificate changed; pair again")
	// ErrLinkButton reports that the bridge link button has not been pressed (Hue error 101).
	ErrLinkButton = errors.New("press bridge button")
	// ErrThrottled reports a bridge rate-limit response (HTTP 429/503).
	ErrThrottled = errors.New("bridge busy")
)

// identity is the unauthenticated bridge identity and its observed leaf certificate.
type identity struct {
	BridgeID    string
	Fingerprint string
	Name        string // Bridge name, software and API version are diagnostics only.
	Software    string
	APIVersion  string
}

// pinnedTLS trusts exactly the pinned leaf certificate. Bridges use either a
// self-signed certificate or one from a private Signify CA, so chain
// verification cannot establish identity; the fingerprint recorded at pairing
// does instead. An empty pin accepts any certificate and reports it to seen.
func pinnedTLS(pin string, seen func(string)) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // Replaced by the explicit leaf pin in VerifyConnection.
		MinVersion:         tls.VersionTLS12,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("bridge presented no certificate")
			}
			sum := sha256.Sum256(state.PeerCertificates[0].Raw)
			fingerprint := hex.EncodeToString(sum[:])
			if pin != "" && fingerprint != pin {
				return ErrCertificateChanged
			}
			if seen != nil {
				seen(fingerprint)
			}
			return nil
		},
	}
}

// probe reads the bridge identity without a key or pin.
func probe(ctx context.Context, address string) (identity, error) {
	var result identity
	transport := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   pinnedTLS("", func(fingerprint string) { result.Fingerprint = fingerprint }),
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: requestTimeout}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+address+"/api/0/config", nil)
	if err != nil {
		return identity{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return identity{}, fmt.Errorf("probe %s: %w", address, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return identity{}, fmt.Errorf("probe %s: HTTP %d", address, response.StatusCode)
	}
	var config struct {
		BridgeID   string `json:"bridgeid"`
		Name       string `json:"name"`
		Software   string `json:"swversion"`
		APIVersion string `json:"apiversion"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&config); err != nil {
		return identity{}, fmt.Errorf("probe %s: %w", address, err)
	}
	if config.BridgeID == "" {
		return identity{}, fmt.Errorf("probe %s: not a Hue bridge", address)
	}
	result.BridgeID = strings.ToLower(config.BridgeID)
	result.Name, result.Software, result.APIVersion = config.Name, config.Software, config.APIVersion
	return result, nil
}

// requestKey asks the bridge for an application key. It returns ErrLinkButton
// until the link button is pressed.
func requestKey(ctx context.Context, address, pin, deviceType string) (string, error) {
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: pinnedTLS(pin, nil)}, Timeout: requestTimeout}
	defer client.CloseIdleConnections()
	body, err := json.Marshal(map[string]string{"devicetype": deviceType})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+address+"/api", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("pair: %w", err)
	}
	defer response.Body.Close()
	var results []struct {
		Success *struct {
			Username string `json:"username"`
		} `json:"success"`
		Error *struct {
			Type        int    `json:"type"`
			Description string `json:"description"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&results); err != nil {
		return "", fmt.Errorf("pair: %w", err)
	}
	for _, result := range results {
		if result.Success != nil && result.Success.Username != "" {
			return result.Success.Username, nil
		}
		if result.Error != nil && result.Error.Type == 101 {
			return "", ErrLinkButton
		}
		if result.Error != nil {
			return "", fmt.Errorf("pair: %s", result.Error.Description)
		}
	}
	return "", errors.New("pair: empty bridge response")
}

// Client issues authenticated CLIP v2 requests to one pinned bridge.
type Client struct {
	base     string
	key      string
	requests *http.Client
	stream   *http.Client
}

func newClient(address, pin, key string) *Client {
	transport := &http.Transport{TLSClientConfig: pinnedTLS(pin, nil), MaxIdleConnsPerHost: 2}
	return &Client{
		base:     "https://" + address,
		key:      key,
		requests: &http.Client{Transport: transport, Timeout: requestTimeout},
		stream:   &http.Client{Transport: transport},
	}
}

func (c *Client) close() { c.requests.CloseIdleConnections() }

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("hue-application-key", c.key)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return c.requests.Do(request)
}

type apiErrors []struct {
	Description string `json:"description"`
}

func (e apiErrors) err() error {
	if len(e) == 0 {
		return nil
	}
	messages := make([]string, 0, len(e))
	for _, item := range e {
		messages = append(messages, item.Description)
	}
	return errors.New(strings.Join(messages, "; "))
}

// Resources loads every CLIP v2 resource.
func (c *Client) Resources(ctx context.Context) ([]resource, error) {
	response, err := c.do(ctx, http.MethodGet, "/clip/v2/resource", nil)
	if err != nil {
		return nil, fmt.Errorf("load resources: %w", err)
	}
	defer response.Body.Close()
	var payload struct {
		Errors apiErrors         `json:"errors"`
		Data   []json.RawMessage `json:"data"`
	}
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("load resources: application key rejected; pair again")
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("load resources: HTTP %d: %w", response.StatusCode, err)
	}
	if err := payload.Errors.err(); err != nil {
		return nil, fmt.Errorf("load resources: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load resources: HTTP %d", response.StatusCode)
	}
	items, err := decodeResources(payload.Data)
	if err != nil {
		return nil, fmt.Errorf("load resources: %w", err)
	}
	return items, nil
}

// Put sends a partial resource update. A rate-limit response wraps ErrThrottled.
func (c *Client) Put(ctx context.Context, kind, id string, body any) error {
	response, err := c.do(ctx, http.MethodPut, "/clip/v2/resource/"+kind+"/"+id, body)
	if err != nil {
		return fmt.Errorf("update %s: %w", kind, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("update %s: %w", kind, ErrThrottled)
	}
	var payload struct {
		Errors apiErrors `json:"errors"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&payload)
	if err := payload.Errors.err(); err != nil {
		return fmt.Errorf("update %s: %w", kind, err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("update %s: HTTP %d", kind, response.StatusCode)
	}
	if decodeErr != nil {
		return fmt.Errorf("update %s: %w", kind, decodeErr)
	}
	return nil
}

// event is one CLIP v2 event-stream notification.
type event struct {
	Type string     `json:"type"`
	Data []resource `json:"data"`
}

// Events follows the bridge event stream, calling handle for each message,
// until the stream ends or ctx is canceled.
func (c *Client) Events(ctx context.Context, handle func([]event) error) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/eventstream/clip/v2", nil)
	if err != nil {
		return err
	}
	request.Header.Set("hue-application-key", c.key)
	request.Header.Set("Accept", "text/event-stream")
	response, err := c.stream.Do(request)
	if err != nil {
		return fmt.Errorf("event stream: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("event stream: HTTP %d", response.StatusCode)
	}
	return readEvents(response.Body, handle)
}

// decodeEvents decodes one event-stream message with tolerant resource decoding.
func decodeEvents(data []byte) ([]event, error) {
	var raw []struct {
		Type string            `json:"type"`
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	events := make([]event, 0, len(raw))
	for _, item := range raw {
		resources, err := decodeResources(item.Data)
		if err != nil {
			return nil, err
		}
		events = append(events, event{Type: item.Type, Data: resources})
	}
	return events, nil
}

// readEvents parses server-sent events whose data fields carry JSON event arrays.
func readEvents(body io.Reader, handle func([]event) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	var data strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if data.Len() == 0 {
				continue
			}
			events, err := decodeEvents([]byte(data.String()))
			if err != nil {
				return fmt.Errorf("event stream: %w", err)
			}
			data.Reset()
			if err := handle(events); err != nil {
				return err
			}
			continue
		}
		if value, ok := strings.CutPrefix(line, "data:"); ok {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(value, " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("event stream: %w", err)
	}
	return io.EOF
}
