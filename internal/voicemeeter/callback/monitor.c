#include "pass_through.h"
#include <mmsystem.h>
#pragma comment(lib, "winmm.lib")

static int timer_requested, power_changed;
static PROCESS_POWER_THROTTLING_STATE previous_power;

/* Registration owner only; never called on the audio thread. Background timer
   throttling batches Remote API callbacks and breaks capture/render pairing. */
static long release_timing(void) {
    long result = 0;
    if (power_changed) {
        if (!SetProcessInformation(GetCurrentProcess(), ProcessPowerThrottling,
                                   &previous_power, sizeof(previous_power))) result = -15;
        else power_changed = 0;
    }
    if (timer_requested) {
        if (timeEndPeriod(1) != TIMERR_NOERROR) result = -14;
        else timer_requested = 0;
    }
    return result;
}

static long acquire_timing(void) {
    PROCESS_POWER_THROTTLING_STATE power = {0};
    long result = release_timing();
    if (result) return result;
    if (timeBeginPeriod(1) != TIMERR_NOERROR) return -14;
    timer_requested = 1;
    previous_power.Version = PROCESS_POWER_THROTTLING_CURRENT_VERSION;
    if (!GetProcessInformation(GetCurrentProcess(), ProcessPowerThrottling,
                               &previous_power, sizeof(previous_power))) {
        if (GetLastError() == ERROR_INVALID_PARAMETER) return 0; /* Older Windows. */
        release_timing(); return -15;
    }
    power = previous_power;
    power.ControlMask |= PROCESS_POWER_THROTTLING_IGNORE_TIMER_RESOLUTION;
    power.StateMask &= ~PROCESS_POWER_THROTTLING_IGNORE_TIMER_RESOLUTION;
    if (!SetProcessInformation(GetCurrentProcess(), ProcessPowerThrottling, &power, sizeof(power))) {
        if (GetLastError() == ERROR_INVALID_PARAMETER) return 0;
        release_timing(); return -15;
    }
    power_changed = 1;
    return 0;
}

static Call stop_callback, unregister_callback;
static int registered;

__declspec(dllexport) long __stdcall SnooferStop(void) {
    long stopped, unregistered, timing;
    if (!registered) return release_timing();
    stopped = stop_callback();
    unregistered = unregister_callback();
    if (unregistered != 0 && unregistered != 1) return unregistered;
    registered = 0;
    timing = release_timing();
    return stopped ? stopped : timing;
}

/* SnooferSetInsert installs (or, with nulls, removes) the insert stages used
   by the next SnooferStart. It refuses while registered: the audio thread
   reads the stages without synchronization. */
__declspec(dllexport) long __stdcall SnooferSetInsert(InsertStage on_input, InsertStage on_output, void *context) {
    if (registered) return -10;
    if (!on_input != !on_output) return -13;
    input_stage = on_input;
    output_stage = on_output;
    stage_context = context;
    return 0;
}

/* SnooferStart registers the output insert, and the input insert too while
   insert stages are set. */
__declspec(dllexport) long __stdcall SnooferStart(HMODULE remote) {
    Register reg;
    Call start;
    char name[64] = "Sound Snoofer audio health";
    long code;
    if (registered) return -10;
    reg = (Register)(void *)GetProcAddress(remote, "VBVMR_AudioCallbackRegister");
    start = (Call)(void *)GetProcAddress(remote, "VBVMR_AudioCallbackStart");
    stop_callback = (Call)(void *)GetProcAddress(remote, "VBVMR_AudioCallbackStop");
    unregister_callback = (Call)(void *)GetProcAddress(remote, "VBVMR_AudioCallbackUnregister");
    if (!reg || !start || !stop_callback || !unregister_callback) return -11;
    code = acquire_timing();
    if (code) return code;
    code = reg(input_stage ? 3 : 2, observe, NULL, name);
    if (code != 0) {
        long timing = release_timing();
        return timing ? timing : code; /* Never unregister another application's slot. */
    }
    registered = 1;
    code = start();
    if (code != 0) {
        if (SnooferStop() != 0) return -12;
        return code;
    }
    return 0;
}

/* Fixed seven uint32 fields; reads occur on the owner's worker, not audio thread. */
__declspec(dllexport) void __stdcall SnooferRead(unsigned long *values) {
    values[0] = (unsigned long)READ(buffers);
    values[1] = (unsigned long)READ(synced);
    values[2] = (unsigned long)READ(starting);
    values[3] = (unsigned long)READ(ending);
    values[4] = (unsigned long)READ(change);
    values[5] = (unsigned long)READ(invalid);
    values[6] = (unsigned long)READ(unknown);
}
