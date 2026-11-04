// Disposable API regressions, linked to the exact libraries used by ai-worker.
// No models, originals, paths or raw output are emitted. Not an exploit or fuzz certification.
#include <opencv2/core.hpp>
#include <opencv2/core/ocl.hpp>
#include <opencv2/dnn.hpp>
#include <opencv2/dnn/all_layers.hpp>
#include <climits>
#include <cmath>
#include <iostream>
#include <stdexcept>
#include <vector>

static std::string stage = "start";
static void require(bool value) { if (!value) throw std::runtime_error("regression"); }
static void nary(int count, bool planes, bool scalar, const char* operation) {
    std::vector<cv::Mat> inputs;
    const int rows = planes ? 8 : 1, cols = 17;
    for (int i = 0; i < count; ++i) {
        const int r = planes && i % 2 ? 1 : rows;
        const int c = scalar && i == count - 1 ? 1 : cols;
        cv::Mat m(r, c, CV_32F);
        for (int y = 0; y < r; ++y) for (int x = 0; x < c; ++x)
            m.at<float>(y,x) = float(i % 4 + 1) + x * .25f;
        inputs.push_back(m);
    }
    cv::dnn::LayerParams params; params.set("operation", operation);
    auto layer = cv::dnn::NaryEltwiseLayer::create(params);
    std::vector<cv::dnn::MatShape> shapes, outShapes, internalShapes;
    for (const auto& m : inputs) shapes.emplace_back(m.size.p, m.size.p + m.dims);
    layer->getMemoryShapes(shapes, 1, outShapes, internalShapes);
    require(outShapes.size() == 1 && outShapes[0] == cv::dnn::MatShape({rows,cols}));
    std::vector<cv::Mat> outputs{cv::Mat(rows, cols, CV_32F)}, internals;
    for (const auto& shape : internalShapes) internals.emplace_back(shape, CV_32F);
    layer->finalize(inputs, outputs); layer->forward(inputs, outputs, internals);
    for (int y = 0; y < rows; ++y) for (int x = 0; x < cols; ++x) {
        float expected = std::string(operation) == "min" ? 1e9f : 0.f;
        for (const auto& m : inputs) {
            const float v = m.at<float>(m.rows == 1 ? 0 : y, m.cols == 1 ? 0 : x);
            if (std::string(operation) == "max") expected = std::max(expected, v);
            else if (std::string(operation) == "min") expected = std::min(expected, v);
            else expected += v;
        }
        if (std::string(operation) == "mean") expected /= count;
        require(std::isfinite(outputs[0].at<float>(y,x)) && std::abs(outputs[0].at<float>(y,x) - expected) < .002f);
    }
}
static cv::MatExpr scalarExpression(const cv::Mat& m, double value) { return m.mul(value); }
static void overwriteStack() { volatile double scratch[256]; for (int i=0;i<256;++i) scratch[i]=-1.; }
int main() {
    cv::setNumThreads(1); cv::ocl::setUseOpenCL(false);
    try {
        int cases = 0;
        // Larger operand counts exercise the pointer-allocation regression.
        // Mixed row/scalar broadcasting also exercises both plane/stride branches.
        for (int count : {3,4,32,160}) for (bool planes : {false,true})
            for (bool scalar : {false,true}) for (auto op : {"sum","mean","max","min"}) {
                stage = "nary-" + std::to_string(count) + "-planes-" + std::to_string(int(planes)) + "-scalar-" + std::to_string(int(scalar)) + "-" + op;
                nary(count, planes, scalar, op); ++cases;
            }
        stage = "u16-saturation";
        uint16_t values[] = {5000,60000,5000,60000,5000,60000};
        cv::Mat m(1,6,CV_16U,values), product = m.mul(m);
        for (int i=0;i<6;++i) require(product.at<uint16_t>(0,i)==65535);
        for (int depth : {CV_8U,CV_16U,CV_32S,CV_32F,CV_64F}) {
            stage = "count-depth-" + std::to_string(depth);
            cv::Mat c(3,257,depth,cv::Scalar(2)); c.row(1).setTo(0);
            require(cv::countNonZero(c)==514);
        }
        stage = "point-dot-overflow";
        require(cv::Point(INT_MIN,INT_MIN).dot(cv::Point(INT_MIN,INT_MIN))==INT_MAX);
        require(cv::Point(INT_MAX,INT_MIN).dot(cv::Point(INT_MAX,INT_MAX))==-2147483647);
        require(cv::Point3i(INT_MIN,INT_MIN,INT_MIN).dot(cv::Point3i(INT_MIN,INT_MIN,INT_MIN))==INT_MAX);
        require(cv::Point3i(INT_MIN,INT_MAX,0).dot(cv::Point3i(INT_MAX,INT_MAX,0))==-2147483647);
        stage = "scalar-lifetime";
        cv::Mat base(2,3,CV_32F,cv::Scalar(3));
        cv::MatExpr (*volatile expression)(const cv::Mat&,double)=scalarExpression;
        void (*volatile overwrite)()=overwriteStack;
        auto delayed=expression(base,7.); overwrite(); cv::Mat result=delayed;
        require(cv::norm(result,cv::Mat(2,3,CV_32F,cv::Scalar(21)),cv::NORM_INF)==0);
        std::cout << "{\"passed\":true,\"nary_cases\":" << cases << ",\"u16_saturation\":true,\"count_depths\":5,\"dot_overflow\":true,\"scalar_lifetime\":true}\n";
        return 0;
    } catch (const cv::Exception& e) { std::cerr << "Native security regression failed: " << stage << " opencv-code=" << e.code << "\n"; return 1; }
    catch (...) { std::cerr << "Native security regression failed: " << stage << "\n"; return 1; }
}
