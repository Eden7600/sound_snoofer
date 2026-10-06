// Windows media sessions for Snoofer. Windows x64 C ABI with UTF-8 JSON
// payloads. Each handle and all of its calls belong to one thread in the
// multithreaded apartment, where blocking on WinRT async results is allowed.
#include <windows.h>
#include <winrt/Windows.Foundation.h>
#include <winrt/Windows.Foundation.Collections.h>
#include <winrt/Windows.Media.Control.h>
#include <winrt/Windows.Storage.Streams.h>

#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <functional>
#include <map>
#include <new>
#include <string>
#include <utility>
#include <vector>

using namespace winrt;
using namespace winrt::Windows::Media::Control;
using namespace winrt::Windows::Storage::Streams;
using Session = GlobalSystemMediaTransportControlsSession;

struct MS {
    GlobalSystemMediaTransportControlsSessionManager manager{nullptr};
    std::vector<std::pair<std::string, Session>> sessions;  // From the last snapshot.
};

namespace {

constexpr uint32_t maxArtBytes = 8 * 1024 * 1024;

void appendJSON(std::string& out, std::string const& s) {
    out += '"';
    for (unsigned char c : s) {
        switch (c) {
        case '"': out += "\\\""; break;
        case '\\': out += "\\\\"; break;
        case '\n': out += "\\n"; break;
        case '\r': out += "\\r"; break;
        case '\t': out += "\\t"; break;
        default:
            if (c < 0x20) {
                char escaped[8];
                std::snprintf(escaped, sizeof escaped, "\\u%04x", c);
                out += escaped;
            } else {
                out += static_cast<char>(c);
            }
        }
    }
    out += '"';
}

int64_t milliseconds(winrt::Windows::Foundation::TimeSpan t) {
    return std::chrono::duration_cast<std::chrono::milliseconds>(t).count();
}

// unixMilliseconds converts a WinRT DateTime (100 ns ticks since 1601).
int64_t unixMilliseconds(winrt::Windows::Foundation::DateTime t) {
    int64_t ticks = t.time_since_epoch().count();
    return ticks == 0 ? 0 : (ticks - 116444736000000000LL) / 10000;
}

const char* statusName(GlobalSystemMediaTransportControlsSessionPlaybackStatus s) {
    switch (s) {
    case GlobalSystemMediaTransportControlsSessionPlaybackStatus::Playing: return "playing";
    case GlobalSystemMediaTransportControlsSessionPlaybackStatus::Paused: return "paused";
    case GlobalSystemMediaTransportControlsSessionPlaybackStatus::Stopped: return "stopped";
    case GlobalSystemMediaTransportControlsSessionPlaybackStatus::Changing: return "changing";
    case GlobalSystemMediaTransportControlsSessionPlaybackStatus::Closed: return "closed";
    default: return "opened";
    }
}

HRESULT deliver(std::string const& data, char* buf, uint32_t cap, uint32_t* needed) {
    if (!needed) return E_INVALIDARG;
    *needed = static_cast<uint32_t>(data.size());
    if (!buf || data.size() > cap) return HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
    std::memcpy(buf, data.data(), data.size());
    return S_OK;
}

Session* find(MS* ms, const char* id) {
    if (!ms || !id) return nullptr;
    for (auto& entry : ms->sessions) {
        if (entry.first == id) return &entry.second;
    }
    return nullptr;
}

void field(std::string& out, const char* name, std::string const& value) {
    out += ",\"";
    out += name;
    out += "\":";
    appendJSON(out, value);
}

void field(std::string& out, const char* name, int64_t value) {
    out += ",\"";
    out += name;
    out += "\":";
    out += std::to_string(value);
}

void flag(std::string& out, const char* name, bool value) {
    out += ",\"";
    out += name;
    out += value ? "\":true" : "\":false";
}

}  // namespace

extern "C" __declspec(dllexport) HRESULT __cdecl MSOpen(MS** out) {
    if (!out) return E_INVALIDARG;
    *out = nullptr;
    try {
        init_apartment(apartment_type::multi_threaded);
    } catch (hresult_error const& e) {
        return e.code();
    }
    try {
        MS* ms = new (std::nothrow) MS;
        if (!ms) {
            uninit_apartment();
            return E_OUTOFMEMORY;
        }
        ms->manager = GlobalSystemMediaTransportControlsSessionManager::RequestAsync().get();
        *out = ms;
        return S_OK;
    } catch (hresult_error const& e) {
        uninit_apartment();
        return e.code();
    }
}

extern "C" __declspec(dllexport) HRESULT __cdecl MSSnapshot(MS* ms, char* buf, uint32_t cap, uint32_t* needed) {
    if (!ms || !needed) return E_INVALIDARG;
    try {
        auto current = ms->manager.GetCurrentSession();
        std::vector<std::pair<std::string, Session>> sessions;
        std::map<std::string, int> seen;
        std::string out = "[";
        for (auto const& s : ms->manager.GetSessions()) {
            std::string app = to_string(s.SourceAppUserModelId());
            int n = seen[app]++;
            std::string id = n == 0 ? app : app + "#" + std::to_string(n);
            std::string title, artist, album;
            bool art = false;
            try {
                auto props = s.TryGetMediaPropertiesAsync().get();
                if (props) {
                    title = to_string(props.Title());
                    artist = to_string(props.Artist());
                    album = to_string(props.AlbumTitle());
                    art = props.Thumbnail() != nullptr;
                }
            } catch (hresult_error const&) {
                // Some players fail metadata requests between tracks; report
                // the session without it.
            }
            auto info = s.GetPlaybackInfo();
            auto controls = info.Controls();
            auto timeline = s.GetTimelineProperties();
            double rate = 1;
            if (auto r = info.PlaybackRate()) rate = r.Value();
            if (out.size() > 1) out += ',';
            out += "{\"id\":";
            appendJSON(out, id);
            field(out, "app", app);
            field(out, "title", title);
            field(out, "artist", artist);
            field(out, "album", album);
            field(out, "status", statusName(info.PlaybackStatus()));
            field(out, "positionMs", milliseconds(timeline.Position() - timeline.StartTime()));
            field(out, "durationMs", milliseconds(timeline.EndTime() - timeline.StartTime()));
            field(out, "updatedMs", unixMilliseconds(timeline.LastUpdatedTime()));
            out += ",\"rate\":" + std::to_string(rate);
            flag(out, "canPlay", controls.IsPlayEnabled());
            flag(out, "canPause", controls.IsPauseEnabled());
            flag(out, "canNext", controls.IsNextEnabled());
            flag(out, "canPrev", controls.IsPreviousEnabled());
            flag(out, "canSeek", controls.IsPlaybackPositionEnabled());
            flag(out, "current", current && current.SourceAppUserModelId() == s.SourceAppUserModelId());
            std::string key;
            if (art) {
                char hash[17];
                std::snprintf(hash, sizeof hash, "%016llx", static_cast<unsigned long long>(std::hash<std::string>{}(title + '\x1f' + artist + '\x1f' + album)));
                key = hash;
            }
            field(out, "artKey", key);
            out += '}';
            sessions.emplace_back(id, s);
        }
        out += ']';
        HRESULT hr = deliver(out, buf, cap, needed);
        if (SUCCEEDED(hr)) ms->sessions = std::move(sessions);
        return hr;
    } catch (hresult_error const& e) {
        return e.code();
    }
}

// MSArt returns the session's thumbnail bytes; S_FALSE with needed 0 when it
// has none.
extern "C" __declspec(dllexport) HRESULT __cdecl MSArt(MS* ms, const char* id, char* buf, uint32_t cap, uint32_t* needed) {
    if (!needed) return E_INVALIDARG;
    Session* s = find(ms, id);
    if (!s) return E_BOUNDS;
    try {
        auto props = s->TryGetMediaPropertiesAsync().get();
        auto thumb = props ? props.Thumbnail() : nullptr;
        if (!thumb) {
            *needed = 0;
            return S_FALSE;
        }
        auto stream = thumb.OpenReadAsync().get();
        uint64_t size = stream.Size();
        if (size == 0 || size > maxArtBytes) {
            *needed = 0;
            return S_FALSE;
        }
        Buffer data(static_cast<uint32_t>(size));
        auto read = stream.ReadAsync(data, static_cast<uint32_t>(size), InputStreamOptions::None).get();
        std::string bytes(reinterpret_cast<const char*>(read.data()), read.Length());
        return deliver(bytes, buf, cap, needed);
    } catch (hresult_error const& e) {
        return e.code();
    }
}

// MSCommand returns S_OK when the player accepted the request and S_FALSE
// when it declined.
extern "C" __declspec(dllexport) HRESULT __cdecl MSCommand(MS* ms, const char* id, int op, int64_t value) {
    Session* s = find(ms, id);
    if (!s) return E_BOUNDS;
    try {
        bool accepted = false;
        switch (op) {
        case 1: accepted = s->TryPlayAsync().get(); break;
        case 2: accepted = s->TryPauseAsync().get(); break;
        case 3: accepted = s->TryTogglePlayPauseAsync().get(); break;
        case 4: accepted = s->TrySkipNextAsync().get(); break;
        case 5: accepted = s->TrySkipPreviousAsync().get(); break;
        case 6: accepted = s->TryChangePlaybackPositionAsync(value * 10000).get(); break;
        default: return E_INVALIDARG;
        }
        return accepted ? S_OK : S_FALSE;
    } catch (hresult_error const& e) {
        return e.code();
    }
}

extern "C" __declspec(dllexport) HRESULT __cdecl MSClose(MS* ms) {
    if (!ms) return E_INVALIDARG;
    ms->sessions.clear();
    ms->manager = nullptr;
    delete ms;
    uninit_apartment();
    return S_OK;
}
