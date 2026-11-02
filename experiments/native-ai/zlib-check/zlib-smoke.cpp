#include <zlib.h>
#include <algorithm>
#include <vector>
#include <string>
#include <iostream>
#include <stdexcept>
static void require(bool value) { if (!value) throw std::runtime_error("zlib operation check failed"); }
int main() {
 try {
  require(std::string(zlibVersion())=="1.3.2" && std::string(ZLIB_VERSION)=="1.3.2");
  std::vector<unsigned char> source(65536), compressed(compressBound(65536)), output(source.size()), copy(source.size());
  for (size_t i=0;i<source.size();++i) source[i]=static_cast<unsigned char>((i*19+i/37)%251);
  uLongf size=static_cast<uLongf>(compressed.size());
  require(compress2(compressed.data(),&size,source.data(),static_cast<uLong>(source.size()),Z_BEST_SPEED)==Z_OK);
  compressed.resize(size);
  uLongf count=static_cast<uLongf>(output.size());
  require(uncompress(output.data(),&count,compressed.data(),size)==Z_OK && count==source.size() && output==source);
  z_stream first{}, second{};
  require(inflateInit(&first)==Z_OK);
  first.next_in=compressed.data();first.avail_in=static_cast<uInt>(compressed.size());first.next_out=output.data();first.avail_out=16;
  require(inflate(&first,Z_NO_FLUSH)==Z_OK && first.total_out==16);
  require(inflateCopy(&second,&first)==Z_OK);
  std::copy(output.begin(),output.begin()+16,copy.begin());
  first.next_out=output.data()+16;first.avail_out=static_cast<uInt>(output.size()-16);
  second.next_out=copy.data()+16;second.avail_out=static_cast<uInt>(copy.size()-16);
  require(inflate(&first,Z_FINISH)==Z_STREAM_END && inflate(&second,Z_FINISH)==Z_STREAM_END);
  require(inflateEnd(&first)==Z_OK && inflateEnd(&second)==Z_OK && output==source && copy==source);
  compressed[0]^=0xff;count=static_cast<uLongf>(output.size());
  require(uncompress(output.data(),&count,compressed.data(),size)==Z_DATA_ERROR);
  std::cout << "{\"zlib\":\"1.3.2\",\"roundtrip\":true,\"inflate_copy\":true,\"malformed_rejected\":true}\n";
  return 0;
 } catch (...) { std::cerr << "zlib operation check failed\n"; return 1; }
}
