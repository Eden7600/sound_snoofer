package hue

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeBridge is an in-process CLIP v2 bridge over TLS.
type fakeBridge struct {
	*httptest.Server
	t *testing.T

	mu          sync.Mutex
	id          string
	key         string
	linkPressed bool
	throttle    int // Number of upcoming PUTs answered with 429.
	ignorePuts  bool
	resources   map[string]resource
	extra       []json.RawMessage // Raw resources in shapes the typed model cannot express.
	puts        []string
	streams     []chan string
}

func newFakeBridge(t *testing.T, id string, items ...resource) *fakeBridge {
	t.Helper()
	b := &fakeBridge{t: t, id: id, key: "secret-key", resources: map[string]resource{}}
	for _, item := range items {
		b.resources[item.ID] = item
	}
	b.Server = httptest.NewTLSServer(http.HandlerFunc(b.serve))
	t.Cleanup(func() {
		b.closeStreams()
		b.Server.Close()
	})
	return b
}

func (b *fakeBridge) address() string { return strings.TrimPrefix(b.URL, "https://") }

func (b *fakeBridge) fingerprint() string {
	sum := sha256.Sum256(b.Certificate().Raw)
	return hex.EncodeToString(sum[:])
}

func (b *fakeBridge) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/0/config":
		fmt.Fprintf(w, `{"name":"Fake","bridgeid":%q,"swversion":"1978293000","apiversion":"1.78.0"}`, strings.ToUpper(b.id))
	case r.Method == http.MethodPost && r.URL.Path == "/api":
		b.mu.Lock()
		pressed := b.linkPressed
		b.mu.Unlock()
		if !pressed {
			fmt.Fprint(w, `[{"error":{"type":101,"address":"","description":"link button not pressed"}}]`)
			return
		}
		fmt.Fprintf(w, `[{"success":{"username":%q}}]`, b.key)
	case r.Header.Get("hue-application-key") != b.key:
		w.WriteHeader(http.StatusForbidden)
	case r.Method == http.MethodGet && r.URL.Path == "/clip/v2/resource":
		b.mu.Lock()
		items := make([]resource, 0, len(b.resources))
		for _, item := range b.resources {
			items = append(items, item)
		}
		data := make([]any, 0, len(items)+len(b.extra))
		for _, item := range items {
			data = append(data, item)
		}
		for _, raw := range b.extra {
			data = append(data, raw)
		}
		b.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"errors": []any{}, "data": data})
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/clip/v2/resource/"):
		b.put(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/eventstream/clip/v2":
		b.stream(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (b *fakeBridge) put(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/clip/v2/resource/"), "/")
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(parts) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b.mu.Lock()
	b.puts = append(b.puts, parts[0]+"/"+parts[1]+":"+canonical(body))
	if b.throttle > 0 {
		b.throttle--
		b.mu.Unlock()
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	ignore := b.ignorePuts
	b.mu.Unlock()
	fmt.Fprint(w, `{"errors":[],"data":[]}`)
	if ignore {
		return
	}
	switch parts[0] {
	case "grouped_light":
		b.applyGroup(parts[1], body)
	case "scene":
		b.update(resource{ID: parts[1], Type: "scene", Status: &sceneStatus{Active: "static"}})
	}
}

func canonical(body map[string]json.RawMessage) string {
	data, _ := json.Marshal(body)
	return string(data)
}

// applyGroup applies a grouped_light write to the group and its lights.
func (b *fakeBridge) applyGroup(id string, body map[string]json.RawMessage) {
	var on *onState
	var dim *dimming
	if raw, ok := body["on"]; ok {
		on = &onState{}
		_ = json.Unmarshal(raw, on)
	}
	if raw, ok := body["dimming"]; ok {
		dim = &dimming{}
		_ = json.Unmarshal(raw, dim)
	}
	b.mu.Lock()
	m := model{resources: b.resources}
	var lights []resource
	for _, r := range b.resources {
		if r.Type != "room" && r.Type != "zone" {
			continue
		}
		for _, s := range r.Services {
			if s.RID == id {
				lights = m.lights(r)
			}
		}
	}
	b.mu.Unlock()
	updates := []resource{{ID: id, Type: "grouped_light", On: on, Dimming: dim}}
	for _, light := range lights {
		updates = append(updates, resource{ID: light.ID, Type: "light", On: on, Dimming: dim})
	}
	b.update(updates...)
}

// update merges resources and publishes them as one update event.
func (b *fakeBridge) update(items ...resource) {
	b.mu.Lock()
	for _, item := range items {
		current := b.resources[item.ID]
		current.merge(item)
		b.resources[item.ID] = current
	}
	streams := append([]chan string(nil), b.streams...)
	b.mu.Unlock()
	data, err := json.Marshal([]event{{Type: "update", Data: items}})
	if err != nil {
		b.t.Error(err)
		return
	}
	for _, s := range streams {
		select {
		case s <- string(data):
		default:
			b.t.Error("fake bridge stream backlog")
		}
	}
}

func (b *fakeBridge) stream(w http.ResponseWriter, r *http.Request) {
	messages := make(chan string, 64)
	b.mu.Lock()
	b.streams = append(b.streams, messages)
	b.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, ": hi\n\n")
	w.(http.Flusher).Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}
			fmt.Fprintf(w, "id: 1:0\ndata: %s\n\n", message)
			w.(http.Flusher).Flush()
		}
	}
}

// pushRaw sends a raw event-stream message.
func (b *fakeBridge) pushRaw(message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.streams {
		s <- message
	}
}

// closeStreams ends every open event stream, simulating a stream gap.
func (b *fakeBridge) closeStreams() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.streams {
		close(s)
	}
	b.streams = nil
}

func (b *fakeBridge) streamCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.streams)
}

func (b *fakeBridge) putLog() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.puts...)
}

func (b *fakeBridge) set(change func(*fakeBridge)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	change(b)
}

// studio returns a room with three lights, a zone and scenes.
func studio() []resource {
	return []resource{
		{ID: "room-1", Type: "room", Metadata: &metadata{Name: "Studio"},
			Children: []reference{{RID: "dev-1", RType: "device"}, {RID: "dev-2", RType: "device"}, {RID: "dev-3", RType: "device"}},
			Services: []reference{{RID: "gl-1", RType: "grouped_light"}}},
		{ID: "zone-1", Type: "zone", Metadata: &metadata{Name: "Desk"},
			Children: []reference{{RID: "light-1", RType: "light"}},
			Services: []reference{{RID: "gl-2", RType: "grouped_light"}}},
		{ID: "dev-1", Type: "device", Services: []reference{{RID: "light-1", RType: "light"}}},
		{ID: "dev-2", Type: "device", Services: []reference{{RID: "light-2", RType: "light"}}},
		{ID: "dev-3", Type: "device", Services: []reference{{RID: "light-3", RType: "light"}}},
		{ID: "gl-1", Type: "grouped_light", On: &onState{On: true}, Dimming: &dimming{Brightness: 50}},
		{ID: "gl-2", Type: "grouped_light", On: &onState{On: true}, Dimming: &dimming{Brightness: 50}},
		{ID: "light-1", Type: "light", On: &onState{On: true}},
		{ID: "light-2", Type: "light", On: &onState{On: true}},
		{ID: "light-3", Type: "light", On: &onState{On: true}},
		{ID: "3f2a9c10-aaaa-bbbb-cccc-000000000001", Type: "scene", Metadata: &metadata{Name: "Bright"},
			Group: &reference{RID: "room-1", RType: "room"}, Status: &sceneStatus{Active: "inactive"}},
		{ID: "8b7e0d21-aaaa-bbbb-cccc-000000000002", Type: "scene", Metadata: &metadata{Name: "Focus"},
			Group: &reference{RID: "zone-1", RType: "zone"}, Status: &sceneStatus{Active: "static"}},
	}
}
