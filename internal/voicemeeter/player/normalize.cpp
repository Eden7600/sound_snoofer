// Offline PCM decoding uses the Windows Media Foundation Source Reader.
// Source files are read-only. The caller owns the temporary output and cancellation event.
#define NOMINMAX
#include <windows.h>
#include <mfapi.h>
#include <mfidl.h>
#include <mfreadwrite.h>
#include <mferror.h>
#include <wrl/client.h>
#include <algorithm>
#include <cmath>
#include <cstdint>
#include <cstdio>
#include <new>
#include <vector>

using Microsoft::WRL::ComPtr;

static bool cancelled(HANDLE event) {
    return WaitForSingleObject(event, 0) == WAIT_OBJECT_0;
}

static HRESULT normalize(const wchar_t* source, const wchar_t* destination, HANDLE cancel) {
    constexpr DWORD stream = static_cast<DWORD>(MF_SOURCE_READER_FIRST_AUDIO_STREAM);
    ComPtr<IMFSourceReader> reader;
    HRESULT hr = MFCreateSourceReaderFromURL(source, nullptr, &reader);
    if (FAILED(hr)) return hr;
    hr = reader->SetStreamSelection(static_cast<DWORD>(MF_SOURCE_READER_ALL_STREAMS), FALSE);
    if (FAILED(hr)) return hr;
    hr = reader->SetStreamSelection(stream, TRUE);
    if (FAILED(hr)) return hr;
    ComPtr<IMFMediaType> requested;
    hr = MFCreateMediaType(&requested);
    if (FAILED(hr)) return hr;
    hr = requested->SetGUID(MF_MT_MAJOR_TYPE, MFMediaType_Audio);
    if (FAILED(hr)) return hr;
    hr = requested->SetGUID(MF_MT_SUBTYPE, MFAudioFormat_PCM);
    if (FAILED(hr)) return hr;
    hr = requested->SetUINT32(MF_MT_AUDIO_BITS_PER_SAMPLE, 16);
    if (FAILED(hr)) return hr;
    hr = reader->SetCurrentMediaType(stream, nullptr, requested.Get());
    if (FAILED(hr)) return hr;
    ComPtr<IMFMediaType> format;
    hr = reader->GetCurrentMediaType(stream, &format);
    if (FAILED(hr)) return hr;
    UINT32 channels = 0, rate = 0, bits = 0;
    if (FAILED(format->GetUINT32(MF_MT_AUDIO_NUM_CHANNELS, &channels)) ||
        FAILED(format->GetUINT32(MF_MT_AUDIO_SAMPLES_PER_SECOND, &rate)) ||
        FAILED(format->GetUINT32(MF_MT_AUDIO_BITS_PER_SAMPLE, &bits)) ||
        channels < 1 || channels > 2 || bits != 16 || rate < 8000 || rate > 192000)
        return MF_E_INVALIDMEDIATYPE;

    std::vector<int16_t> samples;
    int peak = 0;
    for (;;) {
        if (cancelled(cancel)) return HRESULT_FROM_WIN32(ERROR_CANCELLED);
        DWORD flags = 0;
        ComPtr<IMFSample> sample;
        hr = reader->ReadSample(stream, 0, nullptr, &flags, nullptr, &sample);
        if (FAILED(hr)) return hr;
        if (flags & MF_SOURCE_READERF_CURRENTMEDIATYPECHANGED) return MF_E_INVALIDMEDIATYPE;
        if (sample) {
            ComPtr<IMFMediaBuffer> buffer;
            hr = sample->ConvertToContiguousBuffer(&buffer);
            if (FAILED(hr)) return hr;
            DWORD length = 0;
            hr = buffer->GetCurrentLength(&length);
            if (FAILED(hr)) return hr;
            if (length % (channels * 2) || length > 64*1024*1024 ||
                samples.size()*2 + length > 64*1024*1024)
                return HRESULT_FROM_WIN32(ERROR_FILE_TOO_LARGE);
            // Resize before locking so allocation failure cannot strand the buffer lock.
            size_t start = samples.size();
            samples.resize(start + length/2);
            BYTE* data = nullptr;
            hr = buffer->Lock(&data, nullptr, nullptr);
            if (FAILED(hr)) return hr;
            if (length) memcpy(samples.data()+start, data, length);
            hr = buffer->Unlock();
            if (FAILED(hr)) return hr;
            for (size_t n = start; n < samples.size(); ++n)
                peak = std::max(peak, std::abs(static_cast<int>(samples[n])));
        }
        if (flags & MF_SOURCE_READERF_ENDOFSTREAM) break;
    }
    if (samples.empty()) return HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
    // One constant gain preserves the clip's dynamics; silence stays silent.
    const double target = 32768.0 * std::pow(10.0, -1.0/20.0);
    const double gain = peak ? target/peak : 1.0;
    for (size_t n = 0; n < samples.size(); ++n) {
        if (n % 65536 == 0 && cancelled(cancel)) return HRESULT_FROM_WIN32(ERROR_CANCELLED);
        samples[n] = static_cast<int16_t>(std::lround(samples[n]*gain));
    }
    if (cancelled(cancel)) return HRESULT_FROM_WIN32(ERROR_CANCELLED);

    // Windows x64 is little-endian; explicitly lay out the canonical 44-byte PCM header.
    uint8_t header[44] = {};
    auto u16 = [&](int at, uint16_t v) { memcpy(header+at, &v, 2); };
    auto u32 = [&](int at, uint32_t v) { memcpy(header+at, &v, 4); };
    const uint32_t bytes = static_cast<uint32_t>(samples.size()*2);
    memcpy(header, "RIFF", 4); u32(4, 36+bytes); memcpy(header+8, "WAVEfmt ", 8);
    u32(16,16); u16(20,1); u16(22,static_cast<uint16_t>(channels));
    u32(24,rate); u32(28,rate*channels*2);
    u16(32,static_cast<uint16_t>(channels*2)); u16(34,16);
    memcpy(header+36,"data",4); u32(40,bytes);
    FILE* file = nullptr;
    if (_wfopen_s(&file, destination, L"wb") != 0 || !file) return HRESULT_FROM_WIN32(ERROR_OPEN_FAILED);
    bool written = fwrite(header,1,sizeof(header),file) == sizeof(header) &&
        fwrite(samples.data(),1,bytes,file) == bytes;
    if (fclose(file) != 0) written = false;
    if (!written) return HRESULT_FROM_WIN32(ERROR_WRITE_FAULT);
    return cancelled(cancel) ? HRESULT_FROM_WIN32(ERROR_CANCELLED) : S_OK;
}

extern "C" __declspec(dllexport) HRESULT __cdecl SBNormalize(const wchar_t* source, const wchar_t* destination, HANDLE cancel) {
    if (!source || !destination || !cancel || wcscmp(source,destination)==0) return E_INVALIDARG;
    if (cancelled(cancel)) return HRESULT_FROM_WIN32(ERROR_CANCELLED);
    HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
    if (FAILED(hr)) return hr;
    hr = MFStartup(MF_VERSION);
    if (SUCCEEDED(hr)) {
        try { hr = normalize(source,destination,cancel); }
        catch (const std::bad_alloc&) { hr = E_OUTOFMEMORY; }
        HRESULT shutdown = MFShutdown();
        if (SUCCEEDED(hr)) hr = shutdown;
    }
    CoUninitialize();
    return hr;
}

