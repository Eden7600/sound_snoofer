#include <initializer_list>
#include <vector>
#include <fstream>
#include <windows.h>
#include <mmsystem.h>
#include <cassert>
#include <cmath>
#include <cstdio>
#include <cstring>
#include "audio_buffer.h"
struct Config { int mic[2], reference[8], strength, bypass; };
struct Stats { int active, rate, erle, delay; unsigned frames; int failed; };
using Create = HRESULT (__cdecl*)(void**, const char*);
using Destroy = HRESULT (__cdecl*)(void*);
using Configure = HRESULT (__cdecl*)(void*, const Config*);
using Read = HRESULT (__cdecl*)(void*, Stats*);
using Failure = HRESULT (__cdecl*)(void*, int*);
static float samples[8][2048], writes[8][2048], original[8][2048];
int main(int argc, char** argv) {
    assert(argc == 4 || argc == 5);
    bool fullband = argc == 5 && !strcmp(argv[4],"--fullband");
    HMODULE dll = LoadLibraryExA(argv[1], nullptr, LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_DEFAULT_DIRS);
    if (!dll) { fprintf(stderr,"LoadLibrary error %lu\n",GetLastError()); return 1; }
    auto create = reinterpret_cast<Create>(GetProcAddress(dll,fullband ? "AECNeuralFullbandCreate" : "AECNeuralCreate"));
    auto reset = reinterpret_cast<Destroy>(GetProcAddress(dll,"AECResetFailure"));
    auto destroy = reinterpret_cast<Destroy>(GetProcAddress(dll,"AECDestroy"));
    auto configure = reinterpret_cast<Configure>(GetProcAddress(dll,"AECConfigureV2"));
    auto read = reinterpret_cast<Read>(GetProcAddress(dll,"AECReadStats"));
    auto failure = reinterpret_cast<Failure>(GetProcAddress(dll,"AECReadFailure"));
    auto input = reinterpret_cast<InsertStage>(GetProcAddress(dll,"AECInputInsert"));
    auto output = reinterpret_cast<InsertStage>(GetProcAddress(dll,"AECOutputInsert"));
    assert(reset && create && destroy && configure && read && failure && input && output);
    void* e = nullptr;
    assert(FAILED(create(&e,"missing-model.gguf")) && !e);
    assert(SUCCEEDED(create(&e,argv[2])) && e);
    Config c{{2,3},{0,1,2,3,4,5,6,7},0,0};
    assert(SUCCEEDED(configure(e,&c)));
    timeBeginPeriod(1);
    // Repeated warm-up resets used to fill the output queue with results that
    // were never consumed. Test all reset callers before sustained playback.
    {
        AudioBuffer b{};b.sr=48000;b.samples=480;b.inputs=b.outputs=8;
        for(int ch=0;ch<8;++ch){b.read[ch]=samples[ch];b.write[ch]=writes[ch];}
        for(int cycle=0;cycle<30;++cycle){
            if(cycle%3==0)assert(SUCCEEDED(reset(e)));
            else if(cycle%3==1)input(e,nullptr);
            else output(e,&b); // Duplicate output invalidates the previous timeline.
            for(int n=0;n<5;++n){
                for(int ch=0;ch<8;++ch)for(int i=0;i<480;++i)samples[ch][i]=float(ch+1)*.001f;
                input(e,&b);output(e,&b);Sleep(15);
                Stats report{};read(e,&report);
                if(report.failed){
                    int reason=0;failure(e,&reason);
                    fprintf(stderr,"FAIL warm-up reset cycle %d: reason %d\n",cycle,reason);
                    destroy(e);return 1;
                }
                for(int ch=0;ch<8;++ch)assert(!memcmp(samples[ch],writes[ch],480*sizeof(float)));
            }
        }
        assert(SUCCEEDED(reset(e)));
        Stats report{};
        for(int n=0;n<100;++n){input(e,&b);output(e,&b);Sleep(10);read(e,&report);assert(!report.failed);}
        assert(report.active);
        fprintf(stderr,"PASS 30 pre-roll resets and sustained recovery\n");
        input(e,nullptr);
    }
    int rates[] = {16000,32000,48000};
    for (int rate : rates) {
        if(fullband && rate!=48000)continue;
        for (int block : {160,512,1024}) {
            AudioBuffer b{}; b.sr=rate; b.samples=block; b.inputs=b.outputs=8;
            for(int ch=0;ch<8;++ch) { b.read[ch]=samples[ch]; b.write[ch]=writes[ch]; }
            input(e,nullptr);
            // Repeated input pre-roll must stay exact, including in-place buffers.
            for(int pre=0;pre<3;++pre) {
                for(int ch=0;ch<8;++ch) for(int i=0;i<block;++i) samples[ch][i]=float(ch+1)*.01f;
                input(e,&b);
                for(int ch=0;ch<8;++ch) assert(!memcmp(samples[ch],writes[ch],block*sizeof(float)));
            }
            input(e,nullptr);
            Stats s{}; int reason=0;
            uint64_t cursor=0;
            double before=0, after=0;
            int loops=rate*2/block;
            LARGE_INTEGER frequency,start,finish; QueryPerformanceFrequency(&frequency);
            double maxUs=0;
            for(int n=0;n<loops;++n) {
                for(int ch=0;ch<8;++ch) for(int i=0;i<block;++i) {
                    double t=double(cursor+i)/rate;
                    samples[ch][i]=float(.08*sin(2*3.141592653589793*233*t)+.03*sin(2*3.141592653589793*677*t));
                }
                memcpy(original,samples,sizeof(samples));
                // Alternate aliased and distinct callback storage.
                for(int ch=0;ch<8;++ch) b.write[ch]=(n%2 ? samples[ch] : writes[ch]);
                QueryPerformanceCounter(&start); input(e,&b); QueryPerformanceCounter(&finish);
                maxUs=std::fmax(maxUs,double(finish.QuadPart-start.QuadPart)*1e6/frequency.QuadPart);
                assert(SUCCEEDED(read(e,&s)));
                if(s.failed) { failure(e,&reason); fprintf(stderr,"FAIL rate=%d block=%d step=%d reason=%d\n",rate,block,n,reason); return 1; }
                for(int ch=0;ch<8;++ch) for(int i=0;i<block;++i) {
                    assert(std::isfinite(b.write[ch][i]));
                    if(ch!=2 && ch!=3) assert(b.write[ch][i]==original[ch][i]);
                    if(cursor > unsigned(rate) && ch==2) {before+=original[ch][i]*original[ch][i];after+=b.write[ch][i]*b.write[ch][i];}
                }
                // The playback insert uses a distinct bus: its output is never modified.
                memcpy(samples,original,sizeof(samples));
                output(e,&b);
                assert(!memcmp(samples,original,sizeof(samples)));
                cursor+=block;
                Sleep(DWORD(std::ceil(1000.0*block/rate)));
            }
            assert(SUCCEEDED(read(e,&s)) && s.active && s.frames && !s.failed && s.erle==-1 && s.delay==-1);
            printf("PASS %d Hz block %d: active; callback max %.1f us; synthetic attenuation %.1f dB\n",rate,block,maxUs,10*log10((before+1e-20)/(after+1e-20)));
            // A single missing callback must re-prime with exact pass-through.
            input(e,&b); input(e,&b);
            assert(SUCCEEDED(read(e,&s)) && !s.failed && !s.active);
            for(int ch=0;ch<8;++ch) assert(!memcmp(samples[ch],b.write[ch],block*sizeof(float)));
            output(e,&b);
            Sleep(DWORD(std::ceil(1000.0*block/rate)));
            for(int n=0;n<rate/block+3;++n) {
                input(e,&b); output(e,&b);
                assert(SUCCEEDED(read(e,&s)) && !s.failed);
                Sleep(DWORD(std::ceil(1000.0*block/rate)));
            }
            assert(s.active);
            // Extra output also discards history; it cannot be paired with old capture.
            output(e,&b);
            assert(SUCCEEDED(read(e,&s)) && !s.active && !s.failed);
            input(e,nullptr);
            assert(SUCCEEDED(read(e,&s)) && !s.active && !s.failed);
        }
    }
    // Rate/bypass callbacks never corrupt a channel.
    AudioBuffer b{};b.sr=44100;b.samples=512;b.inputs=b.outputs=8;
    for(int ch=0;ch<8;++ch){b.read[ch]=samples[ch];b.write[ch]=writes[ch];}
    input(e,&b);
    for(int ch=0;ch<8;++ch)assert(!memcmp(samples[ch],writes[ch],512*sizeof(float)));
    b.sr=48000; input(e,nullptr);
    Stats deadline{};
    // Drive faster than wall time: late neural results must fail open, not replay.
    for(int n=0;n<200;++n) { input(e,&b); output(e,&b); read(e,&deadline); if(deadline.failed) break; }
    assert(deadline.failed && !deadline.active);
    input(e,nullptr); read(e,&deadline); assert(!deadline.failed && !deadline.active);
    c.bypass=1;assert(SUCCEEDED(configure(e,&c)));b.sr=48000;input(e,&b);
    for(int ch=0;ch<8;++ch)assert(!memcmp(samples[ch],writes[ch],512*sizeof(float)));
    // Frequent gaps cannot restart the recovery budget indefinitely.
    c.bypass=0; assert(SUCCEEDED(configure(e,&c)));
    for(int n=0;n<1500;++n) {
        if(n==400) {
            // 267 ms of valid pairs must clear the previous recovery budget.
            for(int stable=0;stable<25;++stable){input(e,&b);output(e,&b);Sleep(11);}
        }
        input(e,&b);
        if(n%3==0) output(e,&b);
        read(e,&deadline);
        if(n<1200)assert(!deadline.failed);
        if(deadline.failed) break;
        Sleep(11);
    }
    int reason=0;failure(e,&reason);
    assert(deadline.failed && reason==1);
    input(e,&b);output(e,&b);read(e,&deadline);assert(deadline.failed);
    input(e,nullptr);
    // Reference absent altogether still has a five-second input-audio budget.
    for(int n=0;n<480;++n) { input(e,&b);read(e,&deadline);if(n<400)assert(!deadline.failed); }
    read(e,&deadline);failure(e,&reason);assert(deadline.failed && reason==1);
    if(fullband) {
        for(int rate : {16000,32000}) {
            b.sr=rate; input(e,nullptr); input(e,&b); output(e,&b);
            read(e,&deadline);assert(!deadline.active && !deadline.failed);
            for(int ch=0;ch<8;++ch)assert(!memcmp(samples[ch],writes[ch],512*sizeof(float)));
        }
        b.sr=48000;b.samples=480;input(e,nullptr);
        double sine=0,cosine=0;int measured=0;
        for(int n=0;n<300;++n) {
            for(int ch=0;ch<8;++ch)for(int i=0;i<480;++i)samples[ch][i]=float(.04*sin(2*3.141592653589793*10000*(n*480+i)/48000));
            input(e,&b);read(e,&deadline);assert(!deadline.failed);
            if(n>100)for(int i=0;i<480;++i){double phase=2*3.141592653589793*10000*(n*480+i)/48000;sine+=writes[2][i]*sin(phase);cosine+=writes[2][i]*cos(phase);measured++;}
            memset(samples,0,sizeof(samples));output(e,&b);Sleep(10);
        }
        double gain=20*log10(2*sqrt(sine*sine+cosine*cosine)/measured/.04);
        fprintf(stderr,"Full-band callback 10 kHz gain %.2f dB\n",gain);assert(gain>-1 && gain<1);
    }
    if (!fullband) {
    // Compare the complete callback bridge against direct streaming inference on
    // upstream's microphone/reference fixture. This catches accidental muting,
    // reference swaps, sample loss and model-hop alignment errors.
    std::ifstream file(argv[3],std::ios::binary|std::ios::ate);
    assert(file);
    auto bytes=file.tellg(); file.seekg(0);
    std::vector<float> fixture(size_t(bytes)/sizeof(float));
    file.read(reinterpret_cast<char*>(fixture.data()),bytes); assert(file);
    int count=int(fixture.size()/2);
    auto modelDll=GetModuleHandleA("localvqe.dll"); assert(modelDll);
    using NewModel=uintptr_t (__cdecl*)(const char*);
    using Frame=int (__cdecl*)(uintptr_t,const float*,const float*,int,float*);
    using Free=void (__cdecl*)(uintptr_t);
    auto newModel=reinterpret_cast<NewModel>(GetProcAddress(modelDll,"localvqe_new"));
    auto frame=reinterpret_cast<Frame>(GetProcAddress(modelDll,"localvqe_process_frame_f32"));
    auto freeModel=reinterpret_cast<Free>(GetProcAddress(modelDll,"localvqe_free"));
    assert(newModel && frame && freeModel);
    uintptr_t direct=newModel(argv[2]); assert(direct);
    std::vector<float> expected((count+255)/256*256);
    for(int n=0;n<count;n+=256){
        float mic[256]{},ref[256]{};
        for(int i=0;i<256 && n+i<count;++i){mic[i]=fixture[n+i]; for(int ch=0;ch<8;++ch) ref[i]+=fixture[count+n+i]/8.f;}
        assert(frame(direct,mic,ref,256,expected.data()+n)==0);
    }
    freeModel(direct);
    c.bypass=0; assert(SUCCEEDED(configure(e,&c)));
    b.sr=16000;b.samples=160;
    constexpr int latency=1024+160;
    for(int pass=0;pass<2;++pass) {
    if(pass) {
        input(e,&b); input(e,&b); // Recover without a lifecycle/reset call.
        Stats recovering{};read(e,&recovering);assert(!recovering.failed && !recovering.active);
    }
    double maxError=0,energy=0;
    for(int n=0;n<count+latency+160;n+=160){
        for(int ch=0;ch<8;++ch) for(int i=0;i<160;++i)
            samples[ch][i]=n+i<count ? fixture[n+i] : 0;
        input(e,&b);
        Stats report{};read(e,&report);assert(!report.failed);
        for(int i=0;i<160;++i){
            int index=n+i-latency;
            if(index>=0 && index<count){
                maxError=std::fmax(maxError,std::fabs(writes[2][i]-expected[index]));
                energy+=expected[index]*expected[index];
            }
        }
        for(int ch=0;ch<8;++ch) for(int i=0;i<160;++i)
            samples[ch][i]=n+i<count ? fixture[count+n+i] : 0;
        output(e,&b);Sleep(10);
    }
    fprintf(stderr,"Fixture streaming parity: max error %.8f; energy %.6f\n",maxError,energy);
    assert(energy>1e-6 && maxError<1e-4);
    }
    }
    assert(SUCCEEDED(destroy(e)));
    timeEndPeriod(1);
    FreeLibrary(dll);
    puts("PASS neural stream lifecycle and pass-through");
}
