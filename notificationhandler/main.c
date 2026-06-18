#define _GNU_SOURCE
#include <pipewire/pipewire.h>
#include <pipewire/loop.h>
#include <spa/utils/defs.h>
#include <spa/utils/string.h>

#include <curl/curl.h>
#include <jansson.h>

#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#define DEFAULT_URL      "http://time.lucia.friedmicro-lab.org"
#define DEFAULT_BINARY   "Discord"
#define DEFAULT_VOICE    "WEBRTC VoiceEngine"
#define DEFAULT_INTERVAL 120

static volatile sig_atomic_t g_running = 1;
static void on_signal(int sig) { (void)sig; g_running = 0; }

static const char* env_or(const char* name, const char* def) {
    const char* v = getenv(name);
    return (v && *v) ? v : def;
}

static void build_check_url(char* out, size_t cap) {
    const char* base = env_or("TIMEKEEPER_API", DEFAULT_URL);
    snprintf(out, cap, "%s/api/current", base);
}

struct buf {
    char *data;
    size_t len;
    size_t cap;
};

static size_t write_cb(void *ptr, size_t size, size_t nmemb, void *ud) {
    struct buf *b = ud;
    size_t n = size * nmemb;
    if (b->len + n + 1 > b->cap) {
        size_t nc = b->cap ? b->cap * 2 : 4096;
        while (nc < b->len + n + 1)
            nc *= 2;
        char *nd = realloc(b->data, nc);
        if (!nd)
            return 0;
        b->data = nd;
        b->cap = nc;
    }
    memcpy(b->data + b->len, ptr, n);
    b->len += n;
    b->data[b->len] = 0;
    return n;
}

static int check_running(const char *url) {
    CURL *h = curl_easy_init();
    if (!h)
        return -1;
    struct buf b = {0};
    curl_easy_setopt(h, CURLOPT_URL, url);
    curl_easy_setopt(h, CURLOPT_WRITEFUNCTION, write_cb);
    curl_easy_setopt(h, CURLOPT_WRITEDATA, &b);
    curl_easy_setopt(h, CURLOPT_TIMEOUT, 15L);
    curl_easy_setopt(h, CURLOPT_FOLLOWLOCATION, 1L);
    CURLcode rc = curl_easy_perform(h);
    long code = 0;
    curl_easy_getinfo(h, CURLINFO_RESPONSE_CODE, &code);
    curl_easy_cleanup(h);

    if (rc != CURLE_OK) {
        fprintf(stderr, "distraction-killer: GET %s failed: %s\n",
                url, curl_easy_strerror(rc));
        free(b.data);
        return -1;
    }
    if (code != 200) {
        fprintf(stderr, "distraction-killer: GET %s returned HTTP %ld\n", url, code);
        free(b.data);
        return -1;
    }

    json_error_t jerr;
    json_t *root = json_loads(b.data, 0, &jerr);
    free(b.data);
    if (!root) {
        fprintf(stderr, "distraction-killer: JSON parse error: %s\n", jerr.text);
        return -1;
    }
    json_t *run = json_object_get(root, "running");
    int rv = (json_is_boolean(run) && json_boolean_value(run)) ? 1 : 0;
    json_decref(root);
    return rv;
}

typedef struct {
    const char *discord_binary;
    const char *voice_name;
    int found;
    struct pw_registry *registry;
    struct pw_main_loop *loop;
    struct spa_hook **hooks;
    size_t nhooks;
    size_t chooks;
} PwApp;

static int props_match(const PwApp *a, const struct spa_dict *props) {
    const char *binary = spa_dict_lookup(props, PW_KEY_APP_PROCESS_BINARY);
    const char *name = spa_dict_lookup(props, PW_KEY_APP_NAME);
    return binary && name &&
           spa_streq(binary, a->discord_binary) &&
           spa_streq(name, a->voice_name);
}

static void check_info(PwApp *a, const struct spa_dict *props) {
    if (a->found || !props)
        return;
    if (props_match(a, props)) {
        a->found = 1;
        fprintf(stderr,
                "distraction-killer: voice object found (binary=%s name=%s)\n",
                spa_dict_lookup(props, PW_KEY_APP_PROCESS_BINARY),
                spa_dict_lookup(props, PW_KEY_APP_NAME));
    }
}

static void on_node_info(void *data, const struct pw_node_info *info) {
    PwApp *a = data;
    if (info)
        check_info(a, info->props);
    if (a->found)
        pw_main_loop_quit(a->loop);
}

static void on_client_info(void *data, const struct pw_client_info *info) {
    PwApp *a = data;
    if (info)
        check_info(a, info->props);
    if (a->found)
        pw_main_loop_quit(a->loop);
}

static const struct pw_node_events node_events = {
    PW_VERSION_NODE_EVENTS,
    .info = on_node_info,
};

static const struct pw_client_events client_events = {
    PW_VERSION_CLIENT_EVENTS,
    .info = on_client_info,
};

static void keep_hook(PwApp *a, struct spa_hook *h) {
    if (a->nhooks == a->chooks) {
        a->chooks = a->chooks ? a->chooks * 2 : 16;
        a->hooks = realloc(a->hooks, a->chooks * sizeof(struct spa_hook *));
    }
    a->hooks[a->nhooks++] = h;
}

static void registry_global(void *userdata, uint32_t id, uint32_t permissions,
                            const char *type, uint32_t version,
                            const struct spa_dict *props) {
    (void)permissions;
    (void)version;
    (void)props;
    PwApp *a = userdata;

    if (strcmp(type, "PipeWire:Interface:Node") == 0) {
        struct pw_node *node = pw_registry_bind(a->registry, id, type,
                                                PW_VERSION_NODE, 0);
        if (node) {
            struct spa_hook *h = calloc(1, sizeof(*h));
            pw_node_add_listener(node, h, &node_events, a);
            keep_hook(a, h);
        }
    } else if (strcmp(type, "PipeWire:Interface:Client") == 0) {
        struct pw_client *client = pw_registry_bind(a->registry, id, type,
                                                    PW_VERSION_CLIENT, 0);
        if (client) {
            struct spa_hook *h = calloc(1, sizeof(*h));
            pw_client_add_listener(client, h, &client_events, a);
            keep_hook(a, h);
        }
    }
}

static const struct pw_registry_events registry_events = {
    PW_VERSION_REGISTRY_EVENTS,
    .global = registry_global,
};

static const struct pw_core_events core_events = {
    PW_VERSION_CORE_EVENTS,
};

static int pipewire_has_voice(const char *binary, const char *name) {
    PwApp app = {.discord_binary = binary,
                 .voice_name = name,
                 .found = 0,
                 .registry = NULL,
                 .loop = NULL,
                 .hooks = NULL,
                 .nhooks = 0,
                 .chooks = 0};

    struct pw_main_loop *loop = pw_main_loop_new(NULL);
    if (!loop)
        return -1;
    struct pw_context *context = pw_context_new(pw_main_loop_get_loop(loop), NULL, 0);
    if (!context) {
        pw_main_loop_destroy(loop);
        return -1;
    }
    struct pw_core *core = pw_context_connect(context, NULL, 0);
    if (!core) {
        fprintf(stderr, "distraction-killer: pipewire connect failed: %s\n",
                strerror(errno));
        pw_context_destroy(context);
        pw_main_loop_destroy(loop);
        return -1;
    }
    app.loop = loop;

    struct spa_hook cl = {0};
    pw_core_add_listener(core, &cl, &core_events, &app);
    struct pw_registry *registry = pw_core_get_registry(core, PW_VERSION_REGISTRY, 0);
    app.registry = registry;
    struct spa_hook rl = {0};
    pw_registry_add_listener(registry, &rl, &registry_events, &app);

    struct pw_loop *pl = pw_main_loop_get_loop(loop);
    for (int i = 0; i < 30 && !app.found; i++) {
        if (pw_loop_iterate(pl, 200) < 0)
            break;
    }

    spa_hook_remove(&rl);
    spa_hook_remove(&cl);
    pw_proxy_destroy((struct pw_proxy *)registry);
    pw_core_disconnect(core);
    for (size_t i = 0; i < app.nhooks; i++)
        free(app.hooks[i]);
    free(app.hooks);
    pw_context_destroy(context);
    pw_main_loop_destroy(loop);
    return app.found;
}

static void kill_process(const char *binary) {
    char cmd[512];
    snprintf(cmd, sizeof(cmd), "pgrep '%s' | xargs -r kill -9", binary);
    int rc = system(cmd);
    if (rc != 0)
        fprintf(stderr, "distraction-killer: kill command failed (rc=%d)\n", rc);
}

int main(int argc, char **argv) {
    (void)argc;
    (void)argv;

    signal(SIGINT, on_signal);
    signal(SIGTERM, on_signal);

    pw_init(&argc, &argv);
    curl_global_init(CURL_GLOBAL_DEFAULT);

    char check_url[512];
    build_check_url(check_url, sizeof(check_url));

    char discord_binary[128];
    snprintf(discord_binary, sizeof(discord_binary), "%s",
             env_or("DISTRACTION_KILLER_BINARY", DEFAULT_BINARY));

    char voice_name[128];
    snprintf(voice_name, sizeof(voice_name), "%s",
             env_or("DISTRACTION_KILLER_VOICE", DEFAULT_VOICE));

    int poll_interval = DEFAULT_INTERVAL;
    const char* iv = getenv("DISTRACTION_KILLER_INTERVAL");
    if (iv && *iv) {
        long n = strtol(iv, NULL, 10);
        if (n >= 10)
            poll_interval = (int)n;
        else
            fprintf(stderr,
                    "distraction-killer: ignoring DISTRACTION_KILLER_INTERVAL (%ld too small)\n",
                    n);
    }

    fprintf(stderr,
            "distraction-killer started: poll every %ds, url=%s\n"
            "  kill=%s unless voice client (binary=%s name=%s)\n",
            poll_interval, check_url,
            discord_binary, discord_binary, voice_name);

    while (g_running) {
        int st = check_running(check_url);
        if (st == 1) {
            int voice = pipewire_has_voice(discord_binary, voice_name);
            if (voice < 0) {
                fprintf(stderr,
                        "distraction-killer: pipewire check failed; not killing\n");
            } else if (voice == 1) {
                fprintf(stderr,
                        "distraction-killer: Discord voice call active; not killing\n");
            } else {
                fprintf(stderr,
                        "distraction-killer: running and not in a call; killing %s\n",
                        discord_binary);
                kill_process(discord_binary);
            }
        }

        int left = poll_interval;
        while (g_running && left > 0) {
            sleep(1);
            left--;
        }
    }

    curl_global_cleanup();
    pw_deinit();
    fprintf(stderr, "distraction-killer stopped\n");
    return 0;
}
