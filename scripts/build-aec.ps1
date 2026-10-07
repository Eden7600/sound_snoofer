param([string]$OutputDirectory = 'bin')
# Builds bin/snoofer-aec.dll: the vendored WebRTC audio processing (AEC3) and
# its abseil subset, compiled from explicit source lists (no CMake or meson),
# plus the C ABI shim. Third-party objects are cached in .local/aec/lib and
# rebuilt only when a source is newer than the library.
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $w = 'third_party\webrtc-audio-processing\webrtc'
    $a = 'third_party\abseil-cpp\absl'

    # From webrtc's meson files (v2.1) for Windows x64.
    $webrtc = @(
        'rtc_base\checks.cc', 'rtc_base\containers\flat_tree.cc', 'rtc_base\event.cc', 'rtc_base\event_tracer.cc',
        'rtc_base\experiments\field_trial_parser.cc', 'rtc_base\logging.cc', 'rtc_base\memory\aligned_malloc.cc',
        'rtc_base\platform_thread.cc', 'rtc_base\platform_thread_types.cc', 'rtc_base\race_checker.cc', 'rtc_base\random.cc',
        'rtc_base\string_encode.cc', 'rtc_base\string_to_number.cc', 'rtc_base\string_utils.cc', 'rtc_base\strings\string_builder.cc',
        'rtc_base\synchronization\sequence_checker_internal.cc', 'rtc_base\synchronization\yield_policy.cc',
        'rtc_base\system\file_wrapper.cc', 'rtc_base\system_time.cc', 'rtc_base\time_utils.cc', 'rtc_base\zero_memory.cc',
        'api\audio\audio_frame.cc', 'api\audio\audio_processing.cc', 'api\audio\audio_processing_statistics.cc',
        'api\audio\channel_layout.cc', 'api\audio\echo_canceller3_config.cc', 'api\rtp_headers.cc', 'api\rtp_packet_info.cc',
        'api\task_queue\task_queue_base.cc', 'api\units\frequency.cc', 'api\units\time_delta.cc', 'api\units\timestamp.cc',
        'api\video\color_space.cc', 'api\video\hdr_metadata.cc', 'api\video\video_content_type.cc', 'api\video\video_timing.cc',
        'system_wrappers\source\cpu_features.cc', 'system_wrappers\source\denormal_disabler.cc', 'system_wrappers\source\field_trial.cc',
        'system_wrappers\source\metrics.cc', 'system_wrappers\source\sleep.cc',
        'common_audio\audio_converter.cc', 'common_audio\audio_util.cc', 'common_audio\channel_buffer.cc', 'common_audio\fir_filter_c.cc',
        'common_audio\fir_filter_factory.cc', 'common_audio\resampler\push_resampler.cc', 'common_audio\resampler\push_sinc_resampler.cc',
        'common_audio\resampler\resampler.cc', 'common_audio\resampler\sinc_resampler.cc', 'common_audio\resampler\sinusoidal_linear_chirp_source.cc',
        'common_audio\ring_buffer.c', 'common_audio\smoothing_filter.cc', 'common_audio\third_party\ooura\fft_size_128\ooura_fft.cc',
        'common_audio\third_party\ooura\fft_size_256\fft4g.cc', 'common_audio\third_party\spl_sqrt_floor\spl_sqrt_floor.c',
        'common_audio\vad\vad.cc', 'common_audio\vad\vad_core.c', 'common_audio\vad\vad_filterbank.c', 'common_audio\vad\vad_gmm.c',
        'common_audio\vad\vad_sp.c', 'common_audio\vad\webrtc_vad.c',
        'common_audio\fir_filter_sse.cc', 'common_audio\resampler\sinc_resampler_sse.cc', 'common_audio\third_party\ooura\fft_size_128\ooura_fft_sse2.cc',
        'modules\audio_coding\codecs\isac\main\source\filter_functions.c', 'modules\audio_coding\codecs\isac\main\source\isac_vad.c',
        'modules\audio_coding\codecs\isac\main\source\pitch_estimator.c', 'modules\audio_coding\codecs\isac\main\source\pitch_filter.c',
        'third_party\rnnoise\src\rnn_vad_weights.cc'
    )
    foreach ($f in 'auto_correlation', 'auto_corr_to_refl_coef', 'complex_bit_reverse', 'complex_fft', 'copy_set_operations', 'cross_correlation',
        'division_operations', 'downsample_fast', 'filter_ar', 'filter_ar_fast_q12', 'energy', 'filter_ma_fast_q12', 'get_hanning_window',
        'get_scaling_square', 'ilbc_specific_functions', 'levinson_durbin', 'lpc_to_refl_coef', 'min_max_operations', 'randomization_functions',
        'real_fft', 'refl_coef_to_lpc', 'resample_48khz', 'resample_by_2', 'resample_by_2_internal', 'resample', 'resample_fractional',
        'spl_init', 'spl_inl', 'splitting_filter', 'spl_sqrt', 'sqrt_of_one_minus_x_squared', 'vector_scaling_operations') {
        $webrtc += "common_audio\signal_processing\$f.c"
    }
    $webrtc += 'common_audio\signal_processing\dot_product_with_scale.cc'
    $apm = Get-Content -Raw "$w\modules\audio_processing\meson.build"
    $listed = [regex]::Match($apm, "webrtc_audio_processing_sources = \[(.*?)\]", 'Singleline').Groups[1].Value
    foreach ($m in [regex]::Matches($listed, "'([^']+\.cc)'")) { $webrtc += 'modules\audio_processing\' + $m.Groups[1].Value.Replace('/', '\') }
    $webrtc += 'modules\audio_processing\aecm\aecm_core_c.cc'
    $avx2 = @(
        'common_audio\fir_filter_avx2.cc', 'common_audio\resampler\sinc_resampler_avx2.cc',
        'modules\audio_processing\aec3\adaptive_fir_filter_avx2.cc', 'modules\audio_processing\aec3\adaptive_fir_filter_erl_avx2.cc',
        'modules\audio_processing\aec3\fft_data_avx2.cc', 'modules\audio_processing\aec3\matched_filter_avx2.cc',
        'modules\audio_processing\aec3\vector_math_avx2.cc', 'modules\audio_processing\agc2\rnn_vad\vector_math_avx2.cc'
    )
    $pffft = @('third_party\pffft\src\pffft.c')
    $absl = @(
        'base\internal\raw_logging.cc', 'base\internal\throw_delegate.cc', 'base\log_severity.cc',
        'numeric\int128.cc',
        'strings\ascii.cc', 'strings\charconv.cc', 'strings\escaping.cc', 'strings\match.cc', 'strings\numbers.cc',
        'strings\str_cat.cc', 'strings\string_view.cc', 'strings\internal\charconv_bigint.cc', 'strings\internal\charconv_parse.cc',
        'strings\internal\memutil.cc', 'strings\internal\escaping.cc', 'strings\internal\utf8.cc'
    )

    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'MSVC and the Windows SDK are required to build echo cancellation.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    $work = Join-Path $repo '.local/aec'
    New-Item -ItemType Directory -Force $work, $out | Out-Null

    $defines = '/DWEBRTC_WIN /D_WIN32 /D__STDC_FORMAT_MACROS=1 /DNOMINMAX /D_USE_MATH_DEFINES /DWEBRTC_LIBRARY_IMPL /D_WINSOCKAPI_ /DNDEBUG /DWEBRTC_ENABLE_AVX2 /DWEBRTC_APM_DEBUG_DUMP=0 /D_CRT_SECURE_NO_WARNINGS'
    $includes = "/I$w /Ithird_party\abseil-cpp"
    $cxx = "/nologo /c /MP /O2 /MT /EHsc /std:c++20 /Zc:__cplusplus /utf-8 /W0 $defines $includes"
    $lib = Join-Path $work 'webrtc-apm.lib'

    # Rebuild the third-party library only when a vendored source changed.
    $newest = (Get-ChildItem third_party -Recurse -File | Measure-Object LastWriteTime -Maximum).Maximum
    $rebuild = !(Test-Path $lib) -or (Get-Item $lib).LastWriteTime -lt $newest -or (Get-Item $lib).LastWriteTime -lt (Get-Item $PSCommandPath).LastWriteTime
    $lines = @('@echo off', "call `"$vcvars`" >nul", 'if errorlevel 1 exit /b 1')
    # compile emits one cl call per source directory, each into its own object
    # folder, because different directories reuse file names.
    $objects = @()
    $compile = {
        param([string]$root, [string[]]$sources, [string]$extra, [string]$kind)
        foreach ($group in ($sources | Group-Object { Split-Path $_ -Parent })) {
            $dir = ".local\aec\obj\$kind\" + ($group.Name -replace '[\\/]', '_')
            New-Item -ItemType Directory -Force $dir | Out-Null
            $rsp = "$dir.rsp"
            Set-Content -LiteralPath $rsp -Value ($group.Group | ForEach-Object { "$root\$_" })
            $script:lines += "cl $cxx $extra /Fo$dir\ @$rsp || exit /b 1"
            $script:objects += "$dir\*.obj"
        }
    }
    if ($rebuild) {
        if (Test-Path "$work/obj") { Remove-Item -Recurse -Force "$work/obj" }
        & $compile $w $webrtc '' 'webrtc'
        & $compile $w $avx2 '/arch:AVX2' 'avx2'
        & $compile $w $pffft '/D_GNU_SOURCE' 'pffft'
        & $compile $a $absl '' 'absl'
        Set-Content -LiteralPath "$work/objects.rsp" -Value $objects
        $lines += "lib /nologo /out:.local\aec\webrtc-apm.lib @.local\aec\objects.rsp || exit /b 1"
    }
    New-Item -ItemType Directory -Force "$work/obj/shim" | Out-Null
    $lines += "cl /nologo /c /O2 /MT /EHsc /std:c++20 /Zc:__cplusplus /utf-8 /W4 /WX $defines /external:W0 /external:I$w /external:Ithird_party\abseil-cpp /Iinternal\voicemeeter\callback /Fo.local\aec\obj\shim\ internal\aec\native\aec.cpp || exit /b 1"
    $lines += "link /nologo /DLL /OUT:`"$out\snoofer-aec.dll`" /IMPLIB:.local\aec\snoofer-aec.lib .local\aec\obj\shim\aec.obj .local\aec\webrtc-apm.lib winmm.lib || exit /b 1"
    $build = Join-Path $work 'build.cmd'
    Set-Content -LiteralPath $build -Value $lines
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Echo cancellation build failed; do not replace an in-use DLL.' }
} finally { Pop-Location }
