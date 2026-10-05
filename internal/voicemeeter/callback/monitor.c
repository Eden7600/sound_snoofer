#include "pass_through.h"

static Call stop_callback, unregister_callback;
static int registered;

__declspec(dllexport) long __stdcall SnooferStop(void) {
    long stopped, unregistered;
    if (!registered) return 0;
    stopped = stop_callback();
    unregistered = unregister_callback();
    if (unregistered != 0 && unregistered != 1) return unregistered;
    registered = 0;
    return stopped;
}

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
    code = reg(2, observe, NULL, name);
    if (code != 0) return code; /* Never unregister another application's slot. */
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
