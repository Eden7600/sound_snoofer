#include <assert.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>

#ifdef __cplusplus
extern "C" {
#endif
#include "../../internal/voicemeeter/callback/audio_buffer.h"
#ifdef __cplusplus
}
#endif

int main(void) {
    AudioBuffer b = {0};
    float input[] = {0.0f, -0.0f, 0.125f, -1.0f};
    float output[] = {9, 9, 9, 9};
    float untouched[] = {8, 8, 8, 8};
    float overlap[] = {1, 2, 3, 4, 5};
    const float shifted[] = {1, 1, 2, 3, 4};
    const float restored[] = {1, 2, 3, 4, 4};
    const long bad_samples[] = {-1, 0, 65537};
    const long bad_channels[] = {-1, 0, 129};
    size_t n;
    assert(sizeof(long) == 4 && sizeof(void *) == 8);
    assert(offsetof(AudioBuffer, read) == 16 && offsetof(AudioBuffer, write) == 1040);
    assert(sizeof(AudioBuffer) == 2064);
    assert(!audio_buffer_valid_shape(NULL) && !audio_buffer_has_channel(NULL, 0));
    audio_buffer_copy(NULL);
    b.samples = 4;
    b.inputs = b.outputs = 4;
    b.read[0] = input; b.write[0] = output;
    b.write[1] = untouched; /* Missing input. */
    b.read[2] = input; /* Missing output. */
    b.read[3] = b.write[3] = input; /* In place. */
    assert(audio_buffer_valid_shape(&b)); /* Rate is deliberately unspecified. */
    assert(audio_buffer_has_channel(&b, 0) && audio_buffer_has_channel(&b, 3));
    assert(!audio_buffer_has_channel(&b, 1) && !audio_buffer_has_channel(&b, 2));
    assert(!audio_buffer_has_channel(&b, -1) && !audio_buffer_has_channel(&b, 4));
    audio_buffer_copy(&b);
    assert(memcmp(input, output, sizeof input) == 0);
    for (n = 0; n < 4; ++n) assert(untouched[n] == 8);
    assert(input[2] == 0.125f && input[3] == -1.0f);
    for (n = 0; n < sizeof bad_samples / sizeof bad_samples[0]; ++n) {
        b.samples = bad_samples[n];
        assert(!audio_buffer_valid_shape(&b));
        output[0] = 9;
        audio_buffer_copy(&b);
        assert(output[0] == 9);
    }
    b.samples = 65536;
    b.inputs = b.outputs = 128;
    assert(audio_buffer_valid_shape(&b));
    b.samples = 1;
    b.inputs = b.outputs = 1;
    assert(audio_buffer_valid_shape(&b));
    for (n = 0; n < sizeof bad_channels / sizeof bad_channels[0]; ++n) {
        b.inputs = b.outputs = bad_channels[n];
        assert(!audio_buffer_valid_shape(&b) && !audio_buffer_has_channel(&b, 128));
        audio_buffer_copy(&b);
        assert(output[0] == 9);
    }
    b.inputs = 1; b.outputs = 2;
    assert(!audio_buffer_valid_shape(&b));
    audio_buffer_copy(&b);
    assert(output[0] == 9);
    b.samples = 4;
    b.inputs = b.outputs = 1;
    b.read[0] = overlap; b.write[0] = overlap + 1;
    audio_buffer_copy(&b);
    assert(memcmp(overlap, shifted, sizeof overlap) == 0);
    b.read[0] = overlap + 1; b.write[0] = overlap;
    audio_buffer_copy(&b);
    assert(memcmp(overlap, restored, sizeof overlap) == 0);
    puts("PASS: audio buffer shape, channel, alias and overlap checks");
    return 0;
}
