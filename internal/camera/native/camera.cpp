// UVC extension-unit access for Snoofer. Windows x64 C ABI. A handle and all
// of its calls belong to the thread that opened it, which owns a
// multithreaded COM apartment for the handle's lifetime. Only kernel-streaming
// property requests are made; no video stream is ever opened, so other
// applications may use the camera at the same time.
#include <windows.h>
#include <dshow.h>
#include <ks.h>
#include <ksmedia.h>
#include <ksproxy.h>
#include <vidcap.h>

#include <cstring>
#include <cwchar>
#include <new>
#include <string>
#include <vector>

struct CAM {
    IBaseFilter* filter = nullptr;
    IKsControl* control = nullptr;
    ULONG nodes = 0;
    struct Node {
        GUID set;
        ULONG id;
    };
    std::vector<Node> found;  // Property set to node, discovered on first use.
    bool com = false;
};

namespace {

constexpr HRESULT moreData = HRESULT_FROM_WIN32(ERROR_MORE_DATA);
constexpr HRESULT ambiguous = HRESULT_FROM_WIN32(ERROR_DUP_NAME);
constexpr HRESULT noNode = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
constexpr ULONG maxLength = 4096;

void release(CAM* c) {
    if (c->control) c->control->Release();
    if (c->filter) c->filter->Release();
    if (c->com) CoUninitialize();
    delete c;
}

// query sends one KSP_NODE request. With no buffer it asks for the length.
HRESULT query(CAM* c, const GUID& set, ULONG selector, ULONG node, ULONG flags, void* data, ULONG size, ULONG* returned) {
    KSP_NODE p = {};
    p.Property.Set = set;
    p.Property.Id = selector;
    p.Property.Flags = flags | KSPROPERTY_TYPE_TOPOLOGY;
    p.NodeId = node;
    *returned = 0;
    return c->control->KsProperty(reinterpret_cast<PKSPROPERTY>(&p), sizeof(p), data, size, returned);
}

// length asks a node for a control's length. moreData carries the length.
HRESULT length(CAM* c, const GUID& set, ULONG selector, ULONG node, ULONG* size) {
    HRESULT hr = query(c, set, selector, node, KSPROPERTY_TYPE_GET, nullptr, 0, size);
    if (hr == moreData || (SUCCEEDED(hr) && *size > 0)) {
        return *size > 0 && *size <= maxLength ? S_OK : E_UNEXPECTED;
    }
    return FAILED(hr) ? hr : E_UNEXPECTED;
}

// nodeFor finds the extension unit implementing a property set by probing
// each topology node with a length request, and remembers it.
HRESULT nodeFor(CAM* c, const GUID& set, ULONG selector, ULONG* node) {
    for (const auto& n : c->found) {
        if (IsEqualGUID(n.set, set)) {
            *node = n.id;
            return S_OK;
        }
    }
    for (ULONG id = 0; id < c->nodes; ++id) {
        ULONG size = 0;
        if (SUCCEEDED(length(c, set, selector, id, &size))) {
            c->found.push_back({set, id});
            *node = id;
            return S_OK;
        }
    }
    return noNode;
}

bool matches(IMoniker* moniker, const wchar_t* match) {
    IPropertyBag* bag = nullptr;
    if (FAILED(moniker->BindToStorage(nullptr, nullptr, IID_PPV_ARGS(&bag)))) return false;
    VARIANT path;
    VariantInit(&path);
    bool ok = false;
    if (SUCCEEDED(bag->Read(L"DevicePath", &path, nullptr)) && path.vt == VT_BSTR && path.bstrVal) {
        std::wstring lower(path.bstrVal);
        for (auto& ch : lower) ch = static_cast<wchar_t>(towlower(ch));
        ok = lower.find(match) != std::wstring::npos;
    }
    VariantClear(&path);
    bag->Release();
    return ok;
}

}  // namespace

extern "C" {

// CamOpen opens the single video input device whose lowercase device path
// contains match (for example L"vid_2e1a&pid_4c04"). S_FALSE: none present.
// HRESULT_FROM_WIN32(ERROR_DUP_NAME): more than one, which is never guessed.
__declspec(dllexport) HRESULT __cdecl CamOpen(const wchar_t* match, CAM** out) {
    if (!match || !out) return E_INVALIDARG;
    *out = nullptr;
    CAM* c = new (std::nothrow) CAM();
    if (!c) return E_OUTOFMEMORY;
    HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
    if (FAILED(hr)) {
        delete c;
        return hr;
    }
    c->com = true;
    ICreateDevEnum* devices = nullptr;
    IEnumMoniker* monikers = nullptr;
    hr = CoCreateInstance(CLSID_SystemDeviceEnum, nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&devices));
    if (SUCCEEDED(hr)) hr = devices->CreateClassEnumerator(CLSID_VideoInputDeviceCategory, &monikers, 0);
    if (devices) devices->Release();
    if (hr != S_OK) {  // S_FALSE: no video devices at all.
        if (monikers) monikers->Release();
        release(c);
        return FAILED(hr) ? hr : S_FALSE;
    }
    IMoniker* chosen = nullptr;
    int count = 0;
    IMoniker* moniker = nullptr;
    while (monikers->Next(1, &moniker, nullptr) == S_OK) {
        if (matches(moniker, match)) {
            ++count;
            if (!chosen) {
                chosen = moniker;
                continue;
            }
        }
        moniker->Release();
    }
    monikers->Release();
    if (count != 1) {
        if (chosen) chosen->Release();
        release(c);
        return count == 0 ? S_FALSE : ambiguous;
    }
    hr = chosen->BindToObject(nullptr, nullptr, IID_PPV_ARGS(&c->filter));
    chosen->Release();
    if (SUCCEEDED(hr)) hr = c->filter->QueryInterface(IID_PPV_ARGS(&c->control));
    IKsTopologyInfo* topology = nullptr;
    if (SUCCEEDED(hr)) hr = c->filter->QueryInterface(IID_PPV_ARGS(&topology));
    if (SUCCEEDED(hr)) {
        DWORD nodes = 0;
        hr = topology->get_NumNodes(&nodes);
        c->nodes = nodes;
        topology->Release();
    }
    if (FAILED(hr)) {
        release(c);
        return hr;
    }
    *out = c;
    return S_OK;
}

// CamLength reports a control's length in bytes.
__declspec(dllexport) HRESULT __cdecl CamLength(CAM* c, const GUID* set, ULONG selector, ULONG* size) {
    if (!c || !set || !size) return E_INVALIDARG;
    ULONG node = 0;
    HRESULT hr = nodeFor(c, *set, selector, &node);
    if (FAILED(hr)) return hr;
    return length(c, *set, selector, node, size);
}

// CamGet reads a control. *returned is the number of bytes read.
__declspec(dllexport) HRESULT __cdecl CamGet(CAM* c, const GUID* set, ULONG selector, BYTE* data, ULONG size, ULONG* returned) {
    if (!c || !set || !data || !returned || size == 0) return E_INVALIDARG;
    ULONG node = 0;
    HRESULT hr = nodeFor(c, *set, selector, &node);
    if (FAILED(hr)) return hr;
    return query(c, *set, selector, node, KSPROPERTY_TYPE_GET, data, size, returned);
}

// CamSet writes a control. The payload is zero-padded or truncated to the
// length the device reports, which varies with firmware.
__declspec(dllexport) HRESULT __cdecl CamSet(CAM* c, const GUID* set, ULONG selector, const BYTE* data, ULONG size) {
    if (!c || !set || (!data && size)) return E_INVALIDARG;
    ULONG node = 0;
    HRESULT hr = nodeFor(c, *set, selector, &node);
    if (FAILED(hr)) return hr;
    ULONG wanted = 0;
    hr = length(c, *set, selector, node, &wanted);
    if (FAILED(hr)) return hr;
    std::vector<BYTE> payload(wanted, 0);
    memcpy(payload.data(), data, size < wanted ? size : wanted);
    ULONG returned = 0;
    return query(c, *set, selector, node, KSPROPERTY_TYPE_SET, payload.data(), wanted, &returned);
}

// CamClose releases the device and the thread's COM apartment.
__declspec(dllexport) HRESULT __cdecl CamClose(CAM* c) {
    if (c) release(c);
    return S_OK;
}

}  // extern "C"
