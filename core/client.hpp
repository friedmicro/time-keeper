#pragma once

#include "http_backend.hpp"

#include <nlohmann/json.hpp>
#include <stdexcept>
#include <string>
#include <utility>

namespace tk {

template <typename Backend>
class TimeKeeperClient {
public:
    explicit TimeKeeperClient(std::string base, Backend backend = Backend{})
        : base_(std::move(base)), backend_(std::move(backend)) {}

    nlohmann::json get(const std::string& path) {
        HttpResponse r = backend_.get(base_ + path);
        return decode(r);
    }
    nlohmann::json post(const std::string& path, const nlohmann::json& body) {
        HttpResponse r = backend_.post(base_ + path, body.dump());
        return decode(r);
    }

    nlohmann::json weekTimes() { return get("/api/week_times"); }
    nlohmann::json current() { return get("/api/current"); }
    nlohmann::json day(const std::string& date) { return get("/api/day?date=" + date); }
    nlohmann::json start() { return post("/api/start", nlohmann::json::object()); }
    nlohmann::json stop() { return post("/api/stop", nlohmann::json::object()); }
    nlohmann::json addEightHours() { return post("/api/add_8_hours", nlohmann::json::object()); }
    nlohmann::json addTime(double seconds) {
        nlohmann::json body = nlohmann::json::object();
        body["seconds"] = seconds;
        return post("/api/add_time", body);
    }

private:
    static nlohmann::json decode(const HttpResponse& r) {
        if (r.status < 200 || r.status >= 300)
            throw std::runtime_error("TimeKeeper HTTP " + std::to_string(r.status));
        return nlohmann::json::parse(r.body);
    }

public:
    Backend& backend() { return backend_; }

private:
    std::string base_;
    Backend backend_;
};

}