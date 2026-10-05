// Windows x64 C ABI. Each handle and all of its calls belong to one COM thread.
// DirectShow's installed MP3 decoder is used; no default audio renderer is allowed.
#include <windows.h>
#include <dshow.h>
#include <new>
#include <string>

template<class T> static void release(T*& p) { if (p) { p->Release(); p = nullptr; } }

struct Player {
    std::wstring device;
    IGraphBuilder* graph = nullptr;
    IMediaControl* control = nullptr;
    IMediaEvent* events = nullptr;
    HRESULT stop() {
        HRESULT hr = control ? control->Stop() : S_OK;
        release(events);
        release(control);
        release(graph);
        return hr;
    }
    ~Player() { stop(); }
};

static HRESULT renderer(const wchar_t* name, IBaseFilter** result, std::wstring* names) {
    ICreateDevEnum* devices = nullptr;
    IEnumMoniker* enumeration = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_SystemDeviceEnum, nullptr, CLSCTX_INPROC_SERVER,
        IID_PPV_ARGS(&devices));
    if (FAILED(hr)) return hr;
    hr = devices->CreateClassEnumerator(CLSID_AudioRendererCategory, &enumeration, 0);
    devices->Release();
    if (hr != S_OK) return HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
    IMoniker* moniker = nullptr;
    HRESULT found = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
    while (enumeration->Next(1, &moniker, nullptr) == S_OK) {
        IPropertyBag* bag = nullptr;
        if (SUCCEEDED(moniker->BindToStorage(nullptr, nullptr, IID_PPV_ARGS(&bag)))) {
            VARIANT value; VariantInit(&value);
            if (SUCCEEDED(bag->Read(L"FriendlyName", &value, nullptr)) && value.vt == VT_BSTR) {
                if (names) { *names += value.bstrVal; *names += L"\n"; }
                if (name && wcscmp(name, value.bstrVal) == 0) {
                    if (SUCCEEDED(found)) {
                        release(*result);
                        found = HRESULT_FROM_WIN32(ERROR_DUP_NAME);
                        VariantClear(&value); bag->Release(); moniker->Release();
                        break;
                    }
                    found = moniker->BindToObject(nullptr, nullptr, IID_PPV_ARGS(result));
                }
            }
            VariantClear(&value);
            bag->Release();
        }
        moniker->Release();
    }
    enumeration->Release();
    return names ? S_OK : found;
}

extern "C" __declspec(dllexport) HRESULT __cdecl SBDevices(wchar_t* buffer, unsigned capacity) {
    HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
    if (FAILED(hr)) return hr;
    std::wstring names;
    hr = renderer(nullptr, nullptr, &names);
    if (SUCCEEDED(hr)) {
        if (!buffer || names.size() >= capacity) hr = HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
        else wcscpy_s(buffer, capacity, names.c_str());
    }
    CoUninitialize();
    return hr;
}
extern "C" __declspec(dllexport) HRESULT __cdecl SBCreate(const wchar_t* device, Player** out) {
    if (!device || !out) return E_INVALIDARG;
    *out = nullptr;
    HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
    if (FAILED(hr)) return hr;
    Player* player = new(std::nothrow) Player;
    if (!player) { CoUninitialize(); return E_OUTOFMEMORY; }
    player->device = device;
    *out = player;
    return S_OK;
}
extern "C" __declspec(dllexport) HRESULT __cdecl SBStop(Player* player) {
    return player ? player->stop() : E_INVALIDARG;
}
extern "C" __declspec(dllexport) HRESULT __cdecl SBPlay(Player* player, const wchar_t* file, BOOL muted) {
    if (!player || !file) return E_INVALIDARG;
    HRESULT hr = player->stop();
    if (FAILED(hr)) return hr;
    IBaseFilter* output = nullptr;
    IBaseFilter* source = nullptr;
    IFilterGraph2* graph = nullptr;
    IEnumPins* pins = nullptr;
    hr = CoCreateInstance(CLSID_FilterGraph, nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&graph));
    if (SUCCEEDED(hr)) hr = renderer(player->device.c_str(), &output, nullptr);
    if (SUCCEEDED(hr)) hr = graph->AddFilter(output, L"Soundboard output");
    if (SUCCEEDED(hr)) hr = graph->AddSourceFilter(file, L"Clip", &source);
    if (SUCCEEDED(hr)) hr = source->EnumPins(&pins);
    if (SUCCEEDED(hr)) {
        IPin* pin = nullptr;
        bool rendered = false;
        while (pins->Next(1, &pin, nullptr) == S_OK) {
            PIN_DIRECTION direction;
            HRESULT query = pin->QueryDirection(&direction);
            if (SUCCEEDED(query) && direction == PINDIR_OUTPUT) {
                hr = graph->RenderEx(pin, AM_RENDEREX_RENDERTOEXISTINGRENDERERS, nullptr);
                if (SUCCEEDED(hr)) rendered = true;
            }
            pin->Release();
            if (rendered || FAILED(hr)) break;
        }
        if (!rendered && SUCCEEDED(hr)) hr = VFW_E_CANNOT_RENDER;
    }
    release(pins); release(source); release(output);
    if (SUCCEEDED(hr)) hr = graph->QueryInterface(IID_PPV_ARGS(&player->control));
    if (SUCCEEDED(hr)) hr = graph->QueryInterface(IID_PPV_ARGS(&player->events));
    if (SUCCEEDED(hr)) {
        IBasicAudio* audio = nullptr;
        hr = graph->QueryInterface(IID_PPV_ARGS(&audio));
        if (SUCCEEDED(hr)) hr = audio->put_Volume(muted ? -10000 : 0);
        release(audio);
    }
    player->graph = graph;
    if (SUCCEEDED(hr)) hr = player->control->Run();
    if (FAILED(hr)) player->stop();
    return hr;
}
extern "C" __declspec(dllexport) HRESULT __cdecl SBPoll(Player* player) {
    if (!player || !player->events) return S_FALSE;
    long code; LONG_PTR one, two;
    while (player->events->GetEvent(&code, &one, &two, 0) == S_OK) {
        player->events->FreeEventParams(code, one, two);
        if (code == EC_COMPLETE) { HRESULT hr = player->stop(); return FAILED(hr) ? hr : S_FALSE; }
        if (code == EC_ERRORABORT) { player->stop(); return FAILED(static_cast<HRESULT>(one)) ? static_cast<HRESULT>(one) : E_FAIL; }
    }
    return S_OK;
}
extern "C" __declspec(dllexport) HRESULT __cdecl SBClose(Player* player) {
    if (!player) return E_INVALIDARG;
    HRESULT hr = player->stop();
    delete player;
    CoUninitialize();
    return hr;
}

