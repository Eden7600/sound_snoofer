#include <assert.h>
#include <stdio.h>

#include "../../internal/voicemeeter/callback/monitor.c"

static long stop_result, unregister_result;
static int stop_calls, unregister_calls;
static long __stdcall fake_stop(void) { stop_calls++; return stop_result; }
static long __stdcall fake_unregister(void) { unregister_calls++; return unregister_result; }

int main(void) {
    const struct { long stop, unregister, want; int remains_registered; } cases[] = {
        {0, 0, 0, 0}, {0, 1, 0, 0}, {-2, 0, 0, 0}, {-2, 1, 0, 0},
        {-1, 0, -1, 0}, {-3, 1, -3, 0}, {0, -1, -1, 1}, {-2, -1, -1, 1}
    };
    stop_callback = fake_stop;
    unregister_callback = fake_unregister;
    for (size_t n = 0; n < sizeof(cases) / sizeof(cases[0]); n++) {
        registered = 1;
        stop_calls = unregister_calls = 0;
        stop_result = cases[n].stop;
        unregister_result = cases[n].unregister;
        long result = SnooferStop();
        if (result != cases[n].want || registered != cases[n].remains_registered ||
            stop_calls != 1 || unregister_calls != 1) {
            fprintf(stderr, "Stop case %zu: result=%ld registered=%d; want %ld/%d\n",
                    n, result, registered, cases[n].want, cases[n].remains_registered);
            return 1;
        }
        if (!registered) {
            assert(SnooferStop() == 0);
            assert(stop_calls == 1 && unregister_calls == 1);
        }
    }
    registered = 0;
    puts("PASS: clean callback stop, no registration, errors, ownership and repeated stop");
    return 0;
}
