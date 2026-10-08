#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <stddef.h>
#include <assert.h>

#include "../../internal/voicemeeter/callback/pass_through.h"
static volatile LONG cancelled;
static volatile LONG input_count, output_count, repeated_input, repeated_output, last_command, unsynced_input;
static volatile LONG input_rate, input_frames, format_mismatches;
static void (__stdcall *monitor_read)(unsigned long *);
static InsertStage neural_input, neural_output;
static HRESULT (__cdecl *neural_read)(void *, int *);
static HRESULT (__cdecl *neural_timing)(void *, int *);
static void __stdcall observe_neural_input(void *context, AudioBuffer *b) {
    if (b && monitor_read) { InterlockedIncrement(&input_count); if (InterlockedExchange(&last_command,10)==10) InterlockedIncrement(&repeated_input); InterlockedExchange(&input_rate,b->sr); InterlockedExchange(&input_frames,b->samples); }
    float saved[2][2048];
    if (b) {
        if (b->samples > 2048) { pass_through(b); return; }
        for (int c = 0; c < 2; ++c) memcpy(saved[c], b->read[c], (size_t)b->samples * sizeof(float));
    }
    neural_input(context, b);
    /* Restore the observed mic, including aliased read/write buffers. */
    if (b) for (int c = 0; c < 2; ++c) memcpy(b->write[c], saved[c], (size_t)b->samples * sizeof(float));
}
static void __stdcall observe_neural_output(void *context, AudioBuffer *b) {
    if (b && monitor_read) { InterlockedIncrement(&output_count); if (InterlockedExchange(&last_command,11)==11) InterlockedIncrement(&repeated_output); if (b->sr!=READ(input_rate) || b->samples!=READ(input_frames)) InterlockedIncrement(&format_mismatches); }
    neural_output(context,b);
}
static long __stdcall observe_pair(void *user, long command, void *data, long sync) {
    if ((command == 10 || command == 11) && valid((AudioBuffer *)data)) {
        LONG prior = InterlockedExchange(&last_command, command);
        if (command == 10) {
            InterlockedExchange(&input_rate, ((AudioBuffer *)data)->sr);
            InterlockedExchange(&input_frames, ((AudioBuffer *)data)->samples);
            InterlockedIncrement(&input_count);
            if (prior == 10) InterlockedIncrement(&repeated_input);
            if (sync != 1) InterlockedIncrement(&unsynced_input);
        } else {
            if (((AudioBuffer *)data)->sr != READ(input_rate) || ((AudioBuffer *)data)->samples != READ(input_frames)) InterlockedIncrement(&format_mismatches);
            InterlockedIncrement(&output_count);
            if (prior == 11) InterlockedIncrement(&repeated_output);
        }
    } else InterlockedExchange(&last_command, 0);
    return observe(user, command, data, sync);
}

static void snapshot(unsigned long long elapsed) {
    printf("ms=%llu buffers=%ld synced=%ld starting=%ld ending=%ld change=%ld invalid=%ld unknown=%ld sr=%ld frames=%ld channels=%ld\n",
        elapsed, READ(buffers), READ(synced), READ(starting), READ(ending),
        READ(change), READ(invalid), READ(unknown), READ(sample_rate), READ(frame_size), READ(channels));
    printf("input=%ld output=%ld repeated_input=%ld repeated_output=%ld unsynced_input=%ld\n",
        READ(input_count), READ(output_count), READ(repeated_input), READ(repeated_output), READ(unsynced_input));
    printf("input_sr=%ld input_frames=%ld format_mismatches=%ld\n", READ(input_rate), READ(input_frames), READ(format_mismatches));
    if (monitor_read) { unsigned long v[7]; monitor_read(v); printf("monitor buffers=%lu starts=%lu ends=%lu changes=%lu invalid=%lu\n",v[0],v[2],v[3],v[4],v[5]); }
    if (neural_read) {
        int stats[6] = {0}, timing[4] = {0};
        neural_read(stage_context, stats); neural_timing(stage_context, timing);
        printf("neural_active=%d failed=%d frames=%d gaps=%d underruns=%d\n", stats[0], stats[5], stats[4], timing[2], timing[3]);
    }
    fflush(stdout);
}
static int stage_calls[2], stage_resets;
static void __stdcall test_input(void *context, AudioBuffer *b) {
    (void)context;
    if (!b) { stage_resets++; return; }
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
        for (long command = 1; command <= 3; ++command) observe(NULL, command, NULL, 0);
        assert(stage_resets == 3 && stage_calls[0] == 1);
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
    HMODULE dll, neural = NULL, monitor = NULL;
    PROCESS_POWER_THROTTLING_STATE power_before = {PROCESS_POWER_THROTTLING_CURRENT_VERSION, 0, 0};
    int power_known = 0;
    HRESULT (__cdecl *destroy_neural)(void *) = NULL;
    char neural_path[32768], model_path[32768];
    HANDLE guard;
    Call login, logout, start, stop, unregister_callback;
    Register register_callback;
    char name[64] = "Sound Snoofer callback probe";
    ULONGLONG began;
    long code;
    int failed = 0, paired = argc == 3 && wcscmp(argv[1], L"--observe-paired") == 0;
    if (argc == 2 && wcscmp(argv[1], L"--self-test") == 0) return self_test();
    if (argc != 3 || (wcscmp(argv[1], L"--observe") != 0 && !paired)) {
        fputs("Usage: callback-probe --self-test | --observe[|-paired] <absolute installed Remote64.dll path>\n", stderr);
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
    if (paired && GetEnvironmentVariableA("SNOOFER_PROBE_NEURAL", neural_path, sizeof(neural_path))) {
        HRESULT (__cdecl *create)(void **, const char *);
        HRESULT (__cdecl *configure)(void *, const int *);
        int config[12] = {0,1,0,1,2,3,4,5,6,7,0,0};
        if (!GetEnvironmentVariableA("SNOOFER_PROBE_MODEL", model_path, sizeof(model_path))) return 2;
        neural = LoadLibraryExA(neural_path, NULL, LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_DEFAULT_DIRS);
        if (!neural) return 2;
        create = (HRESULT (__cdecl *)(void **, const char *))(void *)GetProcAddress(neural, "AECNeuralCreate");
        configure = (HRESULT (__cdecl *)(void *, const int *))(void *)GetProcAddress(neural, "AECConfigureV2");
        destroy_neural = (HRESULT (__cdecl *)(void *))(void *)GetProcAddress(neural, "AECDestroy");
        neural_input = (InsertStage)(void *)GetProcAddress(neural, "AECInputInsert");
        output_stage = (InsertStage)(void *)GetProcAddress(neural, "AECOutputInsert");
        neural_read = (HRESULT (__cdecl *)(void *, int *))(void *)GetProcAddress(neural, "AECReadStats");
        neural_timing = (HRESULT (__cdecl *)(void *, int *))(void *)GetProcAddress(neural, "AECReadTiming");
        if (!create || !configure || !destroy_neural || !neural_input || !output_stage || !neural_read || !neural_timing) return 2;
        if (FAILED(create(&stage_context, model_path)) || FAILED(configure(stage_context, config))) return 2;
        input_stage = observe_neural_input; neural_output = output_stage; output_stage = observe_neural_output;
    }
    if (neural && GetEnvironmentVariableA("SNOOFER_PROBE_MONITOR", neural_path, sizeof(neural_path))) {
        long (__stdcall *set_insert)(InsertStage, InsertStage, void *);
        long (__stdcall *start_monitor)(HMODULE);
        monitor = LoadLibraryA(neural_path);
        if (!monitor) return 2;
        set_insert = (long (__stdcall *)(InsertStage, InsertStage, void *))(void *)GetProcAddress(monitor,"SnooferSetInsert");
        start_monitor = (long (__stdcall *)(HMODULE))(void *)GetProcAddress(monitor,"SnooferStart");
        stop = (Call)(void *)GetProcAddress(monitor,"SnooferStop");
        monitor_read = (void (__stdcall *)(unsigned long *))(void *)GetProcAddress(monitor,"SnooferRead");
        if (!set_insert || !start_monitor || !stop || !monitor_read) return 2;
        if (set_insert(input_stage,output_stage,stage_context)) return 2;
        power_known = GetProcessInformation(GetCurrentProcess(), ProcessPowerThrottling, &power_before, sizeof(power_before));
        code = start_monitor(dll);
        if (code == 0 && start_monitor(dll) != -10) failed = 1;
    } else {
        puts(paired ? "BEGIN Register paired inserts" : "BEGIN Register output insert"); fflush(stdout);
        code = register_callback(paired ? 3 : 2, paired ? observe_pair : observe, NULL, name);
        printf("END Register code=%ld owner=%.63s\n", code, name); fflush(stdout);
        if (code != 0) { invoke("Logout", logout); return 2; }
        code = invoke("CallbackStart", start);
    }
    if (code == 0) {
        began = GetTickCount64();
        for (int i = 0; i < 20 && !READ(cancelled); ++i) {
            Sleep(500);
            snapshot(GetTickCount64() - began);
            if (READ(invalid) || READ(unknown)) { failed = 1; break; }
        }
    } else failed = 1;
    if (neural_read && !READ(cancelled)) {
        int stats[6] = {0};
        if (FAILED(neural_read(stage_context, stats)) || !stats[0] || stats[5] || !stats[4]) {
            fputs("FAIL: neural bridge did not sustain Active\n", stderr);
            failed = 1;
        }
    }
    if (invoke("CallbackStop", stop) != 0) return 1; /* Do not destroy stages still possibly owned. */
    if (!monitor && invoke("CallbackUnregister", unregister_callback) != 0) return 1;
    snapshot(0);
    if (monitor) {
        PROCESS_POWER_THROTTLING_STATE power_after = {PROCESS_POWER_THROTTLING_CURRENT_VERSION, 0, 0};
        if (stop() != 0) failed = 1;
        if (power_known && (!GetProcessInformation(GetCurrentProcess(), ProcessPowerThrottling, &power_after, sizeof(power_after)) ||
            power_before.ControlMask != power_after.ControlMask || power_before.StateMask != power_after.StateMask)) {
            fputs("FAIL: process timer policy not restored\n", stderr);
            failed = 1;
        }
        if (!failed) puts("PASS: live neural activity, duplicate start refusal, repeated stop, process policy restoration");
    }
    if (neural) { destroy_neural(stage_context); FreeLibrary(neural); }
    if (invoke("Logout", logout) != 0) failed = 1;
    /* Process exit owns DLL release, including uncertain native cleanup. */
    return failed ? 1 : 0;
}
