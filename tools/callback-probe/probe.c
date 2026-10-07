#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <stddef.h>
#include <assert.h>

#include "../../internal/voicemeeter/callback/pass_through.h"
static volatile LONG cancelled;

static void snapshot(unsigned long long elapsed) {
    printf("ms=%llu buffers=%ld synced=%ld starting=%ld ending=%ld change=%ld invalid=%ld unknown=%ld sr=%ld frames=%ld channels=%ld\n",
        elapsed, READ(buffers), READ(synced), READ(starting), READ(ending),
        READ(change), READ(invalid), READ(unknown), READ(sample_rate), READ(frame_size), READ(channels));
    fflush(stdout);
}
static int stage_calls[2];
static void __stdcall test_input(void *context, AudioBuffer *b) {
    (void)context;
    stage_calls[0]++;
    for (long i = 0; i < b->samples; ++i) b->write[0][i] = 0.5f;
}
static void __stdcall test_output(void *context, AudioBuffer *b) {
    (void)b;
    stage_calls[1] += *(int *)context;
}
static int self_test(void) {
    float in[] = {0.0f, -0.0f, 0.125f, -1.0f};
    float out[4] = {0};
    AudioBuffer b = {0};
    assert(sizeof(long) == 4 && sizeof(void *) == 8);
    assert(offsetof(AudioBuffer, read) == 16 && offsetof(AudioBuffer, write) == 1040);
    assert(sizeof(AudioBuffer) == 2064);
    b.sr = 48000; b.samples = 4; b.inputs = b.outputs = 2;
    b.read[0] = in; b.write[0] = out;
    b.read[1] = in; b.write[1] = in;
    observe(NULL, 11, &b, 1);
    assert(memcmp(in, out, sizeof(in)) == 0 && buffers == 1 && synced == 1);
    observe(NULL, 11, &b, 0);
    assert(buffers == 2 && synced == 1);
    b.outputs = 129; observe(NULL, 11, &b, 1);
    b.outputs = 2; b.write[1] = NULL; observe(NULL, 11, &b, 1);
    observe(NULL, 11, NULL, 1);
    assert(invalid == 3 && buffers == 2);
    observe(NULL, 1, NULL, 0); observe(NULL, 2, NULL, 0); observe(NULL, 3, NULL, 0);
    assert(starting == 1 && ending == 1 && change == 1);
    /* Input insert without stages passes through and is not a buffer count. */
    b.write[1] = in;
    memset(out, 0, sizeof(out));
    observe(NULL, 10, &b, 1);
    assert(memcmp(in, out, sizeof(in)) == 0 && buffers == 2 && unknown == 0);
    /* With stages: the input stage owns the input insert's output, and the
       output stage sees the output insert before the usual pass-through. */
    {
        int weight = 1;
        input_stage = test_input;
        output_stage = test_output;
        stage_context = &weight;
        observe(NULL, 10, &b, 1);
        assert(stage_calls[0] == 1 && out[0] == 0.5f && out[3] == 0.5f);
        observe(NULL, 11, &b, 1);
        assert(stage_calls[1] == 1 && memcmp(in, out, sizeof(in)) == 0 && buffers == 3);
        b.write[1] = NULL;
        observe(NULL, 10, &b, 1);
        assert(stage_calls[0] == 1 && invalid == 4);
        b.write[1] = in;
        input_stage = output_stage = NULL;
        stage_context = NULL;
    }
    puts("PASS: ABI, bitwise pass-through, aliasing, sync counters, invalid buffers, lifecycle, insert stages");
    return 0;
}
static BOOL WINAPI control(DWORD event) {
    if (event == CTRL_C_EVENT || event == CTRL_BREAK_EVENT) {
        InterlockedExchange(&cancelled, 1);
        return TRUE;
    }
    return FALSE;
}
static DWORD WINAPI watchdog(void *unused) {
    (void)unused;
    Sleep(25000);
    /* Bound only this probe if a native lifecycle call hangs. Never restart audio. */
    fputs("TIMEOUT: probe exiting; native cleanup unverified\n", stderr);
    fflush(stderr);
    ExitProcess(124);
}
static long invoke(const char *name, Call call) {
    long code;
    printf("BEGIN %s\n", name); fflush(stdout);
    code = call();
    printf("END %s code=%ld\n", name, code); fflush(stdout);
    return code;
}
int wmain(int argc, wchar_t **argv) {
    HMODULE dll;
    HANDLE guard;
    Call login, logout, start, stop, unregister_callback;
    Register register_callback;
    char name[64] = "Sound Snoofer callback probe";
    ULONGLONG began;
    long code;
    int failed = 0;
    if (argc == 2 && wcscmp(argv[1], L"--self-test") == 0) return self_test();
    if (argc != 3 || wcscmp(argv[1], L"--observe") != 0) {
        fputs("Usage: callback-probe --self-test | --observe <absolute installed Remote64.dll path>\n", stderr);
        return 2;
    }
    if (GetFileAttributesW(argv[2]) == INVALID_FILE_ATTRIBUTES) return 2;
    guard = CreateThread(NULL, 0, watchdog, NULL, 0, NULL);
    if (!guard) return 2;
    CloseHandle(guard); /* Thread lives until this short-lived process exits. */
    if (!SetConsoleCtrlHandler(control, TRUE)) return 2;
    dll = LoadLibraryExW(argv[2], NULL, LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_DEFAULT_DIRS);
    if (!dll) { fprintf(stderr, "LoadLibrary error=%lu\n", GetLastError()); return 2; }
#define RESOLVE(variable, type, symbol) variable = (type)(void *)GetProcAddress(dll, symbol); if (!variable) { fputs("Missing " symbol "\n", stderr); return 2; }
    RESOLVE(login, Call, "VBVMR_Login")
    RESOLVE(logout, Call, "VBVMR_Logout")
    RESOLVE(start, Call, "VBVMR_AudioCallbackStart")
    RESOLVE(stop, Call, "VBVMR_AudioCallbackStop")
    RESOLVE(unregister_callback, Call, "VBVMR_AudioCallbackUnregister")
    RESOLVE(register_callback, Register, "VBVMR_AudioCallbackRegister")
    code = invoke("Login", login);
    if (code != 0) { if (code == 1) invoke("Logout", logout); return 2; }
    puts("BEGIN Register output insert"); fflush(stdout);
    code = register_callback(2, observe, NULL, name);
    printf("END Register code=%ld owner=%.63s\n", code, name); fflush(stdout);
    if (code != 0) { invoke("Logout", logout); return 2; }
    code = invoke("CallbackStart", start);
    if (code == 0) {
        began = GetTickCount64();
        for (int i = 0; i < 20 && !READ(cancelled); ++i) {
            Sleep(500);
            snapshot(GetTickCount64() - began);
            if (READ(invalid) || READ(unknown)) { failed = 1; break; }
        }
    } else failed = 1;
    if (invoke("CallbackStop", stop) != 0) failed = 1;
    if (invoke("CallbackUnregister", unregister_callback) != 0) failed = 1;
    snapshot(0);
    if (invoke("Logout", logout) != 0) failed = 1;
    /* Process exit owns DLL release, including uncertain native cleanup. */
    return failed ? 1 : 0;
}
