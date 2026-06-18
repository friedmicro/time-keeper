#pragma once

#include <string>
#include <curl/curl.h>
#include <stdexcept>

namespace tk {

struct HttpResponse {
    long status = 0;
    std::string body;
};

class HttpBackend {
public:
    virtual ~HttpBackend() = default;
    virtual HttpResponse get(const std::string& url) = 0;
    virtual HttpResponse post(const std::string& url, const std::string& body) = 0;
    virtual bool pump() { return false; }
};

class CurlBackend : public HttpBackend {
public:
    CurlBackend() {
        if (curl_global_init(CURL_GLOBAL_DEFAULT) != CURLE_OK)
            throw std::runtime_error("curl_global_init failed");
    }
    ~CurlBackend() override { curl_global_cleanup(); }

    HttpResponse get(const std::string& url) override {
        return perform("GET", url, "");
    }
    HttpResponse post(const std::string& url, const std::string& body) override {
        return perform("POST", url, body);
    }

private:
    static size_t writeCb(char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* out = static_cast<std::string*>(userdata);
        out->append(ptr, size * nmemb);
        return size * nmemb;
    }

    HttpResponse perform(const char* method, const std::string& url, const std::string& body) {
        HttpResponse resp;
        CURL* curl = curl_easy_init();
        if (!curl) throw std::runtime_error("curl_easy_init failed");

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, method);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeCb);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &resp.body);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
        if (method[0] == 'P') {
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body.c_str());
            curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE_LARGE, static_cast<curl_off_t>(body.size()));
            struct curl_slist* headers = nullptr;
            headers = curl_slist_append(headers, "Content-Type: application/json");
            curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
            curl_easy_perform(curl);
            curl_slist_free_all(headers);
        } else {
            curl_easy_perform(curl);
        }

        long code = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &code);
        curl_easy_cleanup(curl);

        resp.status = code;
        return resp;
    }
};

}
