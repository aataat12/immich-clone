// Isolated experimental CPU provider. Source inputs are never rewritten.
#include <opencv2/core.hpp>
#include <opencv2/core/ocl.hpp>
#include <opencv2/dnn.hpp>
#include <opencv2/imgproc.hpp>
#include <opencv2/objdetect/face.hpp>
#include <algorithm>
#include <cmath>
#include <cstdint>
#include <filesystem>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

namespace fs = std::filesystem;
static int quietError(int, const char*, const char*, const char*, int, void*) { return 0; }
static std::vector<unsigned char> readFile(const fs::path& path, size_t limit) {
    std::ifstream f(path, std::ios::binary | std::ios::ate);
    if (!f) throw std::runtime_error("input");
    auto size = f.tellg();
    if (size <= 0 || static_cast<uint64_t>(size) > limit) throw std::runtime_error("size");
    std::vector<unsigned char> b(static_cast<size_t>(size));
    f.seekg(0); f.read(reinterpret_cast<char*>(b.data()), b.size());
    if (!f) throw std::runtime_error("read");
    return b;
}
static uint32_t le32(const unsigned char* p) {
    return uint32_t(p[0]) | uint32_t(p[1]) << 8 | uint32_t(p[2]) << 16 | uint32_t(p[3]) << 24;
}
static cv::Mat frame(const fs::path& path) {
    auto b = readFile(path, 12 + 1600 * 1600 * 3);
    if (b.size() < 12 || std::string(b.begin(), b.begin()+4) != "ICAI") throw std::runtime_error("header");
    uint32_t w = le32(b.data()+4), h = le32(b.data()+8);
    if (w < 32 || h < 32 || w > 1600 || h > 1600 || b.size() != 12 + size_t(w)*h*3) throw std::runtime_error("dimensions");
    return cv::Mat(int(h), int(w), CV_8UC3, b.data()+12).clone();
}
static void faces(const fs::path& models, const cv::Mat& image) {
    auto y = readFile(models / L"yunet.onnx", 45*1024*1024);
    auto s = readFile(models / L"sface.onnx", 45*1024*1024);
    std::vector<unsigned char> config;
    auto detector = cv::FaceDetectorYN::create("onnx", y, config, image.size(), .9f, .3f, 5000,
                                               cv::dnn::DNN_BACKEND_OPENCV, cv::dnn::DNN_TARGET_CPU);
    auto recognizer = cv::FaceRecognizerSF::create("onnx", s, config, cv::dnn::DNN_BACKEND_OPENCV, cv::dnn::DNN_TARGET_CPU);
    // Release serialized models before inference; networks own their parameters.
    y.clear(); y.shrink_to_fit(); s.clear(); s.shrink_to_fit();
    cv::Mat detected; detector->detect(image, detected);
    if (detected.rows > 64) throw std::runtime_error("face_limit");
    std::cout << "{\"protocol\":1,\"faces\":[";
    for (int i=0; i<detected.rows; ++i) {
        auto row = detected.row(i);
        for (int j=0; j<15; ++j) if (!std::isfinite(row.at<float>(0,j))) throw std::runtime_error("face_output");
        cv::Mat aligned, feature;
        recognizer->alignCrop(image, row, aligned); recognizer->feature(aligned, feature);
        if (aligned.size()!=cv::Size(112,112) || feature.total()!=128 || feature.type()!=CV_32F || !cv::checkRange(feature)) throw std::runtime_error("embedding");
        double norm=cv::norm(feature); if (!(norm>0)) throw std::runtime_error("norm");
        feature/=norm;
        if (i) std::cout << ',';
        std::cout << "{\"box\":[" << row.at<float>(0,0) << ',' << row.at<float>(0,1) << ',' << row.at<float>(0,2) << ',' << row.at<float>(0,3)
                  << "],\"confidence\":" << row.at<float>(0,14) << ",\"embedding\":[";
        const float* v=feature.ptr<float>();
        for (int j=0;j<128;++j) { if (j) std::cout << ','; std::cout << v[j]; }
        std::cout << "]}";
    }
    std::cout << "]}\n";
}
struct Object { cv::Rect2d box; float score; int category; };
static void objects(const fs::path& models, const cv::Mat& image) {
    auto weights=readFile(models / L"yolox.onnx",45*1024*1024);
    auto net=cv::dnn::readNetFromONNX(weights);
    weights.clear();weights.shrink_to_fit();
    net.setPreferableBackend(cv::dnn::DNN_BACKEND_OPENCV);net.setPreferableTarget(cv::dnn::DNN_TARGET_CPU);
    double ratio=std::min(640.0/image.cols,640.0/image.rows);
    cv::Mat padded(640,640,CV_32FC3,cv::Scalar(114,114,114)), resized;
    cv::resize(image,resized,cv::Size(int(image.cols*ratio),int(image.rows*ratio)),0,0,cv::INTER_LINEAR);
    resized.convertTo(padded(cv::Rect(0,0,resized.cols,resized.rows)),CV_32F);
    // Raw RGB float32, NCHW, upper-left padding; no normalization by 255.
    net.setInput(cv::dnn::blobFromImage(padded,1.0,cv::Size(),cv::Scalar(),true,false,CV_32F));
    auto output=net.forward();
    if (output.dims!=3 || output.size[0]!=1 || output.size[1]!=8400 || output.size[2]!=85 || output.type()!=CV_32F || !cv::checkRange(output)) throw std::runtime_error("object_output");
    std::vector<Object> candidates; int index=0;
    for (int stride: {8,16,32}) for (int gy=0;gy<640/stride;++gy) for(int gx=0;gx<640/stride;++gx,++index) {
        const float* r=output.ptr<float>(0,index);
        int category=0; for (int c=1;c<80;++c) if (r[5+c]>r[5+category]) category=c;
        float score=r[4]*r[5+category];if (!std::isfinite(score) || score<0 || score>1) throw std::runtime_error("object_score");if (score<.5f) continue;
        double w=std::exp(double(r[2]))*stride,h=std::exp(double(r[3]))*stride;
        double x=(r[0]+gx)*stride-w/2,y=(r[1]+gy)*stride-h/2;
        if (!std::isfinite(w+h+x+y) || !(w>0 && h>0)) throw std::runtime_error("object_box");
        candidates.push_back({cv::Rect2d(x,y,w,h),score,category});
    }
    std::vector<Object> kept;
    for (int c=0;c<80;++c) {
        std::vector<cv::Rect2d> boxes;std::vector<float> scores;std::vector<int> positions,selected;
        for (size_t i=0;i<candidates.size();++i) if (candidates[i].category==c) {boxes.push_back(candidates[i].box);scores.push_back(candidates[i].score);positions.push_back(int(i));}
        cv::dnn::NMSBoxes(boxes,scores,.5f,.5f,selected);
        for (int k:selected) kept.push_back(candidates[positions[k]]);
    }
    if (kept.size()>100) throw std::runtime_error("object_limit");
    std::cout << "{\"protocol\":1,\"objects\":[";bool comma=false;
    for (auto& o:kept) {
        double x=std::clamp(o.box.x/ratio,0.0,double(image.cols)),y=std::clamp(o.box.y/ratio,0.0,double(image.rows));
        double right=std::clamp((o.box.x+o.box.width)/ratio,0.0,double(image.cols)),bottom=std::clamp((o.box.y+o.box.height)/ratio,0.0,double(image.rows));
        if (right<=x || bottom<=y) continue;
        if (comma) std::cout << ',';comma=true;
        std::cout << "{\"class\":" << o.category << ",\"confidence\":" << o.score << ",\"box\":[" << x << ',' << y << ',' << right-x << ',' << bottom-y << "]}";
    }
    std::cout << "]}\n";
}
int wmain(int argc,wchar_t** argv) {
    cv::redirectError(quietError);cv::setNumThreads(1);cv::ocl::setUseOpenCL(false);
    std::cout << std::setprecision(9);
    try {
        if (argc==2 && std::wstring(argv[1])==L"--version") {std::cout << "{\"protocol\":1,\"opencv\":\"" << CV_VERSION << "\",\"runtime\":\"static-native\",\"cpu_baseline\":\"SSE2\"}\n";return 0;}
        if (argc!=4) return 2;
        auto image=frame(fs::path(argv[3]));std::wstring task(argv[1]);
        if (task==L"faces") faces(fs::path(argv[2]),image);
        else if (task==L"objects") objects(fs::path(argv[2]),image);
        else return 2;
        return 0;
    } catch (...) {std::cerr << "AI worker rejected input or failed inference\n";return 1;}
}
