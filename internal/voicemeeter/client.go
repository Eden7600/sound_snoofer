package voicemeeter

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"sound-snoofer/internal/model"
)

var ErrDisconnected = errors.New("Voicemeeter is disconnected; start Banana or Potato")

type native interface {
	Login() int32
	Logout() int32
	Refresh() int32
	Edition() (int32, int32)
	Count(string) int32
	Device(string, int) (model.Device, int32)
	Get(string) (string, int32)
	Set(string, string) int32
	GetNumber(string) (float32, int32)
	SetNumber(string, int) int32
	Release() error
}

// Client owns one login. Calls are serialized; the CLI also pins its OS thread
// because IsParametersDirty must be called from a single thread.
type Client struct {
	processList      func() ([]string, error)
	mu               sync.Mutex
	processorProcess string // Guarded by mu; empty means the default.
	api              native
	closed           bool
	inventory        []model.Device
	inventoryEdition int
	dllPath          string
	login            int32
	version          string
}

func connect(api native) (*Client, error) {
	code := api.Login()
	if code != 0 && code != 1 {
		api.Release()
		return nil, fmt.Errorf("VBVMR_Login returned %d", code)
	}
	return &Client{api: api, login: code}, nil
}

// versioner is implemented by native APIs exposing VBVMR_GetVoicemeeterVersion.
type versioner interface {
	Version() (int32, int32)
}

// RemoteInfo reports the loaded DLL, login result and Voicemeeter version.
// The version is read lazily because it is only available while Voicemeeter runs.
func (c *Client) RemoteInfo() model.RemoteInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.version == "" && !c.closed {
		if v, ok := c.api.(versioner); ok {
			if raw, code := v.Version(); code == 0 && raw != 0 {
				c.version = fmt.Sprintf("%d.%d.%d.%d", raw>>24&0xff, raw>>16&0xff, raw>>8&0xff, raw&0xff)
			}
		}
	}
	return model.RemoteInfo{DLLPath: c.dllPath, Login: c.login, Version: c.version}
}
func status(op string, code int32) error {
	if code == 0 {
		return nil
	}
	if code == -2 {
		return fmt.Errorf("%s: %w", op, ErrDisconnected)
	}
	return fmt.Errorf("%s returned %d", op, code)
}
func (c *Client) refresh() error {
	if c.closed {
		return errors.New("Voicemeeter client is closed")
	}
	code := c.api.Refresh()
	if code < 0 {
		return status("VBVMR_IsParametersDirty", code)
	}
	return nil
}
func (c *Client) Snapshot() (model.Snapshot, error) { return c.snapshot(true) }

// SetProcessorProcess selects the executable reported as the voice processor.
func (c *Client) SetProcessorProcess(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processorProcess = name
}

// ParameterSnapshot reads fresh parameters using the last full inventory.
// Callers retain full observations around transactions and device assignments.
func (c *Client) ParameterSnapshot() (model.Snapshot, error) { return c.snapshot(false) }
func (c *Client) snapshot(enumerate bool) (model.Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	empty := model.Snapshot{}
	if e := c.refresh(); e != nil {
		return empty, e
	}
	edition, code := c.api.Edition()
	if e := status("VBVMR_GetVoicemeeterType", code); e != nil {
		return empty, e
	}
	s := model.Snapshot{Edition: int(edition), Devices: []model.Device{}, Assignments: map[string]string{}}
	if !enumerate && (c.inventory == nil || c.inventoryEdition != int(edition)) {
		return empty, errors.New("full inventory observation required")
	}
	if enumerate {
		for _, direction := range []string{"input", "output"} {
			n := c.api.Count(direction)
			if n < 0 {
				return empty, fmt.Errorf("%s enumeration returned %d", direction, n)
			}
			if n > 4096 {
				return empty, fmt.Errorf("unreasonable %s device count %d", direction, n)
			}
			for i := 0; i < int(n); i++ {
				dev, code := c.api.Device(direction, i)
				if e := status(fmt.Sprintf("%s device %d", direction, i), code); e != nil {
					return empty, e
				}
				dev.Direction = direction
				// Only WDM enumeration is eligibility evidence in this milestone. ASIO
				// entries may describe installed drivers with no connected hardware.
				dev.Available = dev.Driver == "wdm"
				s.Devices = append(s.Devices, dev)
			}
		}
	} else {
		s.Devices = append([]model.Device{}, c.inventory...)
	}
	for _, target := range model.Slots(s.Edition) {
		slot, _ := model.ParseSlot(target)
		name, code := c.api.Get(slot.Parameter("name"))
		if e := status("read "+target, code); e != nil {
			return empty, e
		}
		s.Assignments[target] = name
	}
	if model.StripCount(s.Edition) > 0 {
		s.Numbers = map[string]float32{}
		n, _ := model.Limits(s.Edition)
		// Two ASIO patch cells (left, right) per hardware input.
		params := []string{}
		for cell := 0; cell < 2*n; cell++ {
			params = append(params, fmt.Sprintf("Patch.asio[%d]", cell))
		}
		for strip := 0; strip < model.StripCount(s.Edition); strip++ {
			for bus := 1; bus <= s.Edition; bus++ {
				params = append(params, fmt.Sprintf("Strip[%d].B%d", strip, bus))
			}
			for bus := 1; bus <= n; bus++ {
				params = append(params, fmt.Sprintf("Strip[%d].A%d", strip, bus))
			}
		}
		for _, param := range params {
			v, code := c.api.GetNumber(param)
			if e := status("read "+param, code); e != nil {
				return empty, e
			}
			s.Numbers[param] = v
		}
	}
	// Optional health/mixer observations must not break legacy parameter snapshots.
	for _, target := range model.Slots(s.Edition) {
		slot, _ := model.ParseSlot(target)
		p := slot.Parameter("sr")
		if v, code := c.api.GetNumber(p); code == 0 {
			s.Numbers[p] = v
		}
	}
	for _, prefix := range []string{"Strip", "Bus"} {
		count := model.StripCount(s.Edition)
		if prefix == "Bus" {
			count, _ = model.Limits(s.Edition)
		}
		for j := 0; j < count; j++ {
			for _, suffix := range []string{"Gain", "Mute"} {
				p := fmt.Sprintf("%s[%d].%s", prefix, j, suffix)
				if v, code := c.api.GetNumber(p); code == 0 {
					s.Numbers[p] = v
				}
			}
		}
	}
	if enumerate {
		c.inventory = append([]model.Device{}, s.Devices...)
		c.inventoryEdition = s.Edition
	}
	if c.processList != nil {
		names, err := c.processList()
		processor := c.processorProcess
		if processor == "" {
			processor = "element.exe" // Matches config.DefaultProcessorProcess.
		}
		element := observeProcess(names, err, processor)

		s.Element = &element

	}

	if monitor, ok := c.api.(interface{ CallbackStatus() *model.CallbackStatus }); ok {
		s.Callback = monitor.CallbackStatus()
	}
	return s, nil
}
func (c *Client) Set(target string, device model.Device) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	slot, e := model.ParseSlot(target)
	if e != nil {
		return e
	}
	if (device.Driver != "wdm" && !(device.Driver == "asio" && target == "A1" && device.Name != "")) || device.Direction != slot.Direction || !device.Available {
		return errors.New("invalid device assignment (ASIO is only supported on A1)")
	}
	if e = c.refresh(); e != nil {
		return e
	}
	edition, code := c.api.Edition()
	if e = status("edition before write", code); e != nil {
		return e
	}
	n, e := model.Limits(int(edition))
	if e != nil {
		return e
	}
	if slot.Index >= n {
		return fmt.Errorf("%s unavailable in edition %d", target, edition)
	}
	return status("assign "+target, c.api.Set(slot.Parameter(device.Driver), device.Name))
}

var patchParam = regexp.MustCompile(`^Patch\.asio\[([0-9])\]$`)
var sendParam = regexp.MustCompile(`^Strip\[([0-7])\]\.([AB])([1-5])$`)

func (c *Client) SetNumber(param string, value int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.refresh(); e != nil {
		return e
	}
	edition, code := c.api.Edition()
	if e := status("edition before numeric write", code); e != nil {
		return e
	}
	n, e := model.Limits(int(edition))
	if e != nil {
		return e
	}
	if m := patchParam.FindStringSubmatch(param); m != nil {
		// Two cells per hardware input; the value is an ASIO input channel
		// (1-based) or 0 for none.
		cell, _ := strconv.Atoi(m[1])
		if cell >= 2*n || value < 0 || value > 64 {
			return errors.New("ASIO patch cell or input channel out of range")
		}
	} else if m := sendParam.FindStringSubmatch(param); m != nil {
		strip, _ := strconv.Atoi(m[1])
		bus, _ := strconv.Atoi(m[3])
		if m[2] == "B" {
			n = int(edition)
		}
		if strip >= model.StripCount(int(edition)) || bus > n || (value != 0 && value != 1) {
			return errors.New("invalid strip routing value or target")
		}
	} else {
		return fmt.Errorf("numeric parameter is not managed: %q", param)
	}
	return status("set "+param, c.api.SetNumber(param, value))
}
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	if callback, ok := c.api.(interface {
		SetCallback(bool, *InsertHook) error
	}); ok {
		if err := callback.SetCallback(false, nil); err != nil {
			return err
		}
	}
	c.closed = true
	return errors.Join(status("VBVMR_Logout", c.api.Logout()), c.api.Release())
}
