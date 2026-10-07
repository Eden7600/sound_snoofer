package voicemeeter

import (
	"slices"
	"testing"
)

type closeOrderAPI struct {
	fakeAPI
	calls []string
}

func (a *closeOrderAPI) SetCallback(monitor bool, insert *InsertHook) error {
	if monitor || insert != nil {
		panic("unexpected callback start")
	}
	a.calls = append(a.calls, "stop")
	return nil
}
func (a *closeOrderAPI) Logout() int32  { a.calls = append(a.calls, "logout"); return 0 }
func (a *closeOrderAPI) Release() error { a.calls = append(a.calls, "release"); return nil }

func TestCloseStopsCallbackBeforeLogoutAndRelease(t *testing.T) {
	a := &closeOrderAPI{}
	c, err := connect(a)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.calls, []string{"stop", "logout", "release"}) {
		t.Fatal(a.calls)
	}
}
