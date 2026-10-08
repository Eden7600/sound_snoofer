#include <windows.h>
#include <cassert>
#include <cstdio>
#include <vector>
#include <algorithm>
#include "common_audio/resampler/push_sinc_resampler.h"
#include "../../internal/aec/native/fullband.h"
static void Acoustic(bool echo, bool voice) {
    constexpr int count=48000*20;
    constexpr double pi=3.14159265358979323846;
    std::vector<float> farEnd(count);
    unsigned seed=928;
    for(auto& f:farEnd){seed=seed*1664525u+1013904223u;f=echo ? float(int(seed>>16)-32768)/327680.f : 0;}
    snoofer::Fullband engine;
    snoofer::Crossover dry, wet;
    double inputPower=0,outputPower=0,sinPart=0,cosPart=0;
    int measured=0;
    for(int n=0;n<count;n+=480){
        float mic[480],out[480]{},reference[480];
        for(int i=0;i<480;++i){
            int t=n+i;
            reference[i]=farEnd[t];
            mic[i]=(t>=1920 ? .6f*farEnd[t-1920] : 0)+(t>=2013 ? .3f*farEnd[t-2013] : 0);
            if(voice)mic[i]+=float(.08*sin(2*pi*1100*t/48000)+.04*sin(2*pi*10000*t/48000));
        }
        engine.Push(mic,reference);
        for(int i=0;i<480;++i)out[i]=voice ? float(.08*sin(2*pi*1100*(n+i-834)/48000)) : 0;
        engine.Mix(n,out,480);
        for(int i=0;i<480;++i){
            float raw=dry.Process(0,mic[i]); float upper=wet.Process(0,out[i]);
            if(n<15*48000)continue;
            inputPower+=raw*raw;outputPower+=upper*upper;
            sinPart+=out[i]*sin(2*pi*10000*(n+i)/48000);
            cosPart+=out[i]*cos(2*pi*10000*(n+i)/48000);measured++;
        }
    }
    double reduction=10*log10((inputPower+1e-20)/(outputPower+1e-20));
    double gain=20*log10((2*sqrt(sinPart*sinPart+cosPart*cosPart)/measured+1e-20)/.04);
    printf("Upper band echo=%d voice=%d: reduction %.2f dB, 10 kHz voice gain %.2f dB\n",echo,voice,reduction,gain);
    fflush(stdout);
    if(echo && !voice)assert(reduction>10);
    // AEC3 gates the upper band during tonal double-talk; report that limitation.
    if(voice && !echo)assert(gain>-1 && gain<1);
}
int main() {
    setvbuf(stdout,nullptr,_IONBF,0);
    constexpr int count=48000;
    std::vector<float> source(count),out(count);
    unsigned seed=42;
    for(auto& f:source) { seed=seed*1664525u+1013904223u; f=float(int(seed>>16)-32768)/327680.f; }
    auto apm=snoofer::HighBandAEC();
    webrtc::StreamConfig mono(48000,1);
    for(int n=0;n<count;n+=480) {
        float ref[480]{};float* rp[]={ref};const float* sp[]={source.data()+n};float* op[]={out.data()+n};
        assert(!apm->ProcessReverseStream(rp,mono,mono,rp));
        assert(!apm->ProcessStream(sp,mono,mono,op));
    }
    int best=0;double max=-1;
    for(int delay=0;delay<1200;++delay) {
        double corr=0;for(int i=4800;i<count;++i) corr+=out[i]*source[i-delay];
        if(corr>max){max=corr;best=delay;}
    }
    printf("Measured AEC3 delay: %d samples; correlation %.5f\n",best,max);
    assert(best==snoofer::Fullband::kAECDelay);
    // Pin the other branch's resampler + one-model-hop group delay.
    webrtc::PushSincResampler down(480,160),up(256,768);
    std::fill(out.begin(),out.end(),0.0f);
    float mic16[160],hop[256]{},previous[256]{},resampled[768];
    int filled=0,emitted=0;
    for(int n=0;n<count;n+=480){
        down.Resample(source.data()+n,480,mic16,160);
        for(float v:mic16){hop[filled++]=v;if(filled!=256)continue;
            filled=0;up.Resample(previous,256,resampled,768);
            std::memcpy(previous,hop,sizeof(hop));
            std::copy(resampled,resampled+768,out.begin()+emitted);emitted+=768;
        }
    }
    best=0;max=-1;
    for(int d=0;d<1200;++d){double corr=0;for(int i=4800;i<emitted;++i)corr+=out[i]*source[i-d];if(corr>max){max=corr;best=d;}}
    printf("Measured neural transport delay: %d samples\n",best);assert(best==834);
    snoofer::Crossover split;
    for(int i=0;i<count;++i) assert(split.Process(source[i],source[i])==(i<64?0:source[i-64]));
    puts("PASS: complementary reconstruction");
    for(int frequency : {1000,6000,10000}){
        for(bool lower : {false,true}){
            snoofer::Crossover filter;
            double energy=0;
            for(int i=0;i<48000;++i){float x=float(sin(2*3.14159265358979323846*frequency*i/48000));float y=filter.Process(lower?x:0,lower?0:x);if(i>=4800)energy+=y*y;}
            double gain=10*log10(energy/(43200*.5));
            printf("Crossover %d Hz lower=%d gain %.2f dB\n",frequency,lower,gain);
            if(frequency==6000)assert(gain>-6.1 && gain<-5.9);
            else if((frequency==1000)==lower)assert(gain>-.1 && gain<.1);
            else assert(gain<-60);
        }
    }
    snoofer::UpperGate gate;
    for(int i=0;i<4800;++i)assert(gate.Process(0)==0);
    float gain=0;
    for(int i=0;i<2400;++i)gain=gate.Process(.02f);
    assert(gain>.99f);
    float previousGain=gain;
    for(int i=0;i<48000;++i){gain=gate.Process(0);assert(gain>=0 && gain<=previousGain+.000001f);previousGain=gain;}
    assert(gain<.001f);
    puts("PASS: gate stays closed on silence, opens on neural voice and smoothly releases");
    Acoustic(false,true);Acoustic(true,false);Acoustic(true,true);
}






