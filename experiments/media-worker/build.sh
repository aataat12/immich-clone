#!/usr/bin/env bash
# Experiment only: no application integration or distribution acceptance implied.
set -euo pipefail
commit=1041abdc962f4cc4f394aa8de9dc5236c0c3b9e7
root="$RUNNER_TEMP/minimal-media"
mkdir -p "$root/source" "$GITHUB_WORKSPACE/media-candidate"
cd "$root/source"
git init -q
git remote add origin https://github.com/FFmpeg/FFmpeg.git
git fetch -q --depth 1 origin "$commit"
git checkout -q --detach FETCH_HEAD
test "$(git rev-parse HEAD)" = "$commit"
git archive --prefix=ffmpeg-8.1.3/ HEAD | gzip -n > "$GITHUB_WORKSPACE/media-candidate/ffmpeg-8.1.3-source.tar.gz"
args=(
 --target-os=mingw32 --arch=x86_64 --cpu=x86-64 --enable-cross-compile
 --cross-prefix=x86_64-w64-mingw32- --cc=x86_64-w64-mingw32-gcc-win32
 --pkg-config=false --disable-autodetect --disable-network --disable-doc
 --disable-debug --disable-x86asm --disable-shared --enable-static
 --disable-everything --enable-small --enable-ffmpeg --enable-ffprobe --disable-ffplay
 --disable-avdevice --disable-iconv --disable-bzlib --disable-zlib --disable-lzma
 --enable-decoder=h264,hevc,aac --enable-parser=h264,hevc,aac
 --enable-demuxer=mov,matroska,image2 --enable-protocol=file,pipe
 --enable-encoder=bmp --enable-muxer=image2,image2pipe,null
 --enable-filter=scale,format,transpose,hflip,vflip,xstack,crop
 '--extra-cflags=-O2 -march=x86-64 -mtune=generic -D_WIN32_WINNT=0x0A00'
 --extra-ldflags=-static
)
printf '%q ' ./configure "${args[@]}" > "$GITHUB_WORKSPACE/media-candidate/configure-command.txt"
./configure "${args[@]}" > "$root/configure-output.txt"
cat "$root/configure-output.txt"
make -j2 ffmpeg.exe ffprobe.exe > "$root/build.log" 2>&1 || { tail -80 "$root/build.log"; exit 1; }
cp ffmpeg.exe ffprobe.exe COPYING.LGPLv2.1 LICENSE.md "$GITHUB_WORKSPACE/media-candidate/"
printf '%s\n' "$commit" > "$GITHUB_WORKSPACE/media-candidate/source-commit.txt"
x86_64-w64-mingw32-objdump -p ffmpeg.exe ffprobe.exe | sed -n 's/.*DLL Name: /DLL: /p' | sort -u > "$GITHUB_WORKSPACE/media-candidate/pe-imports.txt"
cat "$GITHUB_WORKSPACE/media-candidate/pe-imports.txt"
dpkg-query --show gcc-mingw-w64-x86-64-win32 binutils-mingw-w64-x86-64 mingw-w64-common mingw-w64-x86-64-dev > "$GITHUB_WORKSPACE/media-candidate/build-tool-versions.txt"
cp /usr/share/doc/gcc-mingw-w64-x86-64-win32/copyright "$GITHUB_WORKSPACE/media-candidate/compiler-copyright.txt"
cp /usr/share/doc/mingw-w64-common/copyright "$GITHUB_WORKSPACE/media-candidate/mingw-copyright.txt"
cd "$GITHUB_WORKSPACE/media-candidate"
sha256sum ffmpeg.exe ffprobe.exe ffmpeg-8.1.3-source.tar.gz > SHA256SUMS.txt
