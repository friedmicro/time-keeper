#include "client.hpp"

#include <nlohmann/json.hpp>

#include <gtk/gtk.h>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <ctime>
#include <functional>
#include <memory>
#include <string>
#include <thread>
#include <utility>

namespace {

const char* const kBase = [] {
    const char* env = getenv("TIMEKEEPER_API");
    return env && *env ? env : "http://time.lucia.friedmicro-lab.org";
}();
const int kRefreshSeconds = 10;

struct App {
    GtkWidget* status_label = nullptr;
    GtkWidget* display_label = nullptr;
    GtkWidget* category_entry = nullptr;
    GtkWidget* start_btn = nullptr;
    GtkWidget* stop_btn = nullptr;

    std::atomic<bool> running{false};
    std::atomic<long long> started_epoch{0};
    std::atomic<bool> busy{false};
};

App* app_state = nullptr;

using Callback = std::function<void()>;

gboolean idle_trampoline(gpointer data) {
    auto* cb = static_cast<Callback*>(data);
    (*cb)();
    delete cb;
    return G_SOURCE_REMOVE;
}

void run_on_main(Callback cb) {
    auto* p = new Callback(std::move(cb));
    g_idle_add(idle_trampoline, p);
}

static long long tm_to_epoch_utc(struct tm *tm) {
    const char* prev = getenv("TZ");
    std::string saved = prev ? std::string(prev) : "";
    setenv("TZ", "UTC", 1);
    tzset();
    long long epoch = static_cast<long long>(mktime(tm));
    if (saved.empty())
        unsetenv("TZ");
    else
        setenv("TZ", saved.c_str(), 1);
    tzset();
    return epoch;
}

long long parse_wallclock(const std::string& s) {
    if (s.empty()) return 0;

    bool utc = !s.empty() && (s.back() == 'Z');

    struct tm t{};
    int y = 0, mo = 0, d = 0, h = 0, mi = 0, se = 0;
    if (sscanf(s.c_str(), "%d-%d-%dT%d:%d:%d", &y, &mo, &d, &h, &mi, &se) != 6)
        return 0;

    t.tm_year = y - 1900;
    t.tm_mon = mo - 1;
    t.tm_mday = d;
    t.tm_hour = h;
    t.tm_min = mi;
    t.tm_sec = se;
    t.tm_isdst = -1;

    if (utc) {
        return tm_to_epoch_utc(&t);
    } else {
        const char* prev = getenv("TZ");
        std::string saved = prev ? std::string(prev) : "";
        setenv("TZ", "America/Los_Angeles", 1);
        tzset();
        long long epoch = static_cast<long long>(mktime(&t));
        if (saved.empty())
            unsetenv("TZ");
        else
            setenv("TZ", saved.c_str(), 1);
        tzset();
        return epoch;
    }
}

std::string format_hms(long long seconds) {
    if (seconds < 0) seconds = 0;
    char buf[16];
    snprintf(buf, sizeof(buf), "%02lld:%02lld:%02lld",
             seconds / 3600, (seconds % 3600) / 60, seconds % 60);
    return buf;
}

void set_running_ui(bool running) {
    gtk_widget_set_sensitive(app_state->start_btn, !running);
    gtk_widget_set_sensitive(app_state->stop_btn, running);
    gtk_label_set_label(GTK_LABEL(app_state->status_label),
                        running ? "Running" : "Stopped");
    gtk_widget_set_name(app_state->status_label, running ? "running" : "");
}

void refresh_display() {
    if (app_state->running.load()) {
        long long now = static_cast<long long>(time(nullptr));
        long long el = now - app_state->started_epoch.load();
        gtk_label_set_label(GTK_LABEL(app_state->display_label),
                            format_hms(el).c_str());
    } else {
        gtk_label_set_label(GTK_LABEL(app_state->display_label), "00:00:00");
    }
}

void apply_current(const nlohmann::json& data) {
    bool running = data.value("running", false);
    app_state->running.store(running);
    if (running) {
        std::string started = data.value("started_at", "");
        if (!started.empty()) {
            long long ep = parse_wallclock(started);
            app_state->started_epoch.store(ep);
        }
    }
    set_running_ui(running);
    refresh_display();
}

void refresh_from_worker() {
    try {
        tk::TimeKeeperClient<tk::CurlBackend> client(kBase);
        auto data = client.current();
        run_on_main([data]() { apply_current(data); });
    } catch (const std::exception& e) {
        run_on_main([msg = std::string(e.what())]() {
            gtk_label_set_label(GTK_LABEL(app_state->status_label),
                                ("error: " + msg).c_str());
        });
    }
}

void start_from_worker(const std::string& category) {
    try {
        tk::TimeKeeperClient<tk::CurlBackend> client(kBase);
        nlohmann::json body = nlohmann::json::object();
        if (!category.empty()) body["category"] = category;
        client.post("/api/start", body);
        refresh_from_worker();
    } catch (const std::exception& e) {
        run_on_main([msg = std::string(e.what())]() {
            gtk_label_set_label(GTK_LABEL(app_state->status_label),
                                ("start failed: " + msg).c_str());
            app_state->busy.store(false);
        });
    }
}

void stop_from_worker() {
    try {
        tk::TimeKeeperClient<tk::CurlBackend> client(kBase);
        client.post("/api/stop", nlohmann::json::object());
        refresh_from_worker();
    } catch (const std::exception& e) {
        run_on_main([msg = std::string(e.what())]() {
            gtk_label_set_label(GTK_LABEL(app_state->status_label),
                                ("stop failed: " + msg).c_str());
            app_state->busy.store(false);
        });
    }
}

void kick_refresh() {
    if (app_state->busy.exchange(true)) return;
    std::thread t([]() {
        refresh_from_worker();
        app_state->busy.store(false);
    });
    t.detach();
}

void on_refresh(GtkWidget*, gpointer) {
    kick_refresh();
}

void on_start(GtkWidget*, gpointer) {
    if (app_state->busy.exchange(true)) return;
    const char* text = gtk_editable_get_text(GTK_EDITABLE(app_state->category_entry));
    std::string category = text ? text : "";
    std::thread t([category]() {
        start_from_worker(category);
        app_state->busy.store(false);
    });
    t.detach();
}

void on_stop(GtkWidget*, gpointer) {
    if (app_state->busy.exchange(true)) return;
    std::thread t([]() {
        stop_from_worker();
        app_state->busy.store(false);
    });
    t.detach();
}

gboolean on_tick(gpointer) {
    refresh_display();
    return G_SOURCE_CONTINUE;
}

gboolean on_periodic_refresh(gpointer) {
    kick_refresh();
    return G_SOURCE_CONTINUE;
}

GtkWidget* make_box(GtkOrientation orient, int spacing) {
    GtkWidget* box = gtk_box_new(orient, spacing);
    gtk_widget_set_halign(box, GTK_ALIGN_CENTER);
    gtk_widget_set_valign(box, GTK_ALIGN_CENTER);
    return box;
}

void activate(GtkApplication* app, gpointer) {
    app_state = new App();

    GtkWidget* win = gtk_application_window_new(app);
    gtk_window_set_title(GTK_WINDOW(win), "TimeKeeper");
    gtk_window_set_default_size(GTK_WINDOW(win), 360, 260);

    GtkWidget* vbox = make_box(GTK_ORIENTATION_VERTICAL, 8);
    gtk_window_set_child(GTK_WINDOW(win), vbox);

    app_state->status_label = gtk_label_new("Stopped");
    gtk_label_set_xalign(GTK_LABEL(app_state->status_label), 0.5f);
    gtk_box_append(GTK_BOX(vbox), app_state->status_label);

    app_state->display_label = gtk_label_new("00:00:00");
    PangoAttrList* attrs = pango_attr_list_new();
    pango_attr_list_insert(attrs, pango_attr_size_new_absolute(48 * PANGO_SCALE));
    gtk_label_set_attributes(GTK_LABEL(app_state->display_label), attrs);
    pango_attr_list_unref(attrs);
    gtk_box_append(GTK_BOX(vbox), app_state->display_label);

    app_state->category_entry = gtk_entry_new();
    gtk_entry_set_placeholder_text(GTK_ENTRY(app_state->category_entry),
                                   "What are you working on?");
    gtk_box_append(GTK_BOX(vbox), app_state->category_entry);

    GtkWidget* hbox = make_box(GTK_ORIENTATION_HORIZONTAL, 8);
    app_state->start_btn = gtk_button_new_with_label("Start");
    app_state->stop_btn = gtk_button_new_with_label("Stop");
    g_signal_connect(app_state->start_btn, "clicked", G_CALLBACK(on_start), nullptr);
    g_signal_connect(app_state->stop_btn, "clicked", G_CALLBACK(on_stop), nullptr);
    gtk_box_append(GTK_BOX(hbox), app_state->start_btn);
    gtk_box_append(GTK_BOX(hbox), app_state->stop_btn);
    gtk_box_append(GTK_BOX(vbox), hbox);

    gtk_widget_set_visible(win, true);

    set_running_ui(false);
    on_refresh(nullptr, nullptr);
    g_timeout_add_seconds(1, on_tick, nullptr);
    g_timeout_add_seconds(kRefreshSeconds, on_periodic_refresh, nullptr);
}

}

int main(int argc, char** argv) {
    GtkApplication* app = gtk_application_new("org.friedmicro.tk", G_APPLICATION_DEFAULT_FLAGS);
    g_signal_connect(app, "activate", G_CALLBACK(activate), nullptr);
    int status = g_application_run(G_APPLICATION(app), argc, argv);
    g_object_unref(app);
    return status;
}
