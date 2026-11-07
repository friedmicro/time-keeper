use serde::Deserialize;
use std::io::{BufRead, BufReader, Read, Write};
use std::os::unix::net::UnixStream;

const DAEMON_SOCKET: &str = "/tmp/waybar-damon";
const DAEMON_KEY: &str = "timekeeper";

#[derive(Deserialize)]
struct StatusResponse {
    break_met: bool,
    quota_met: bool,
    today_hours: f64,
    totals: f64,
}

#[derive(serde::Serialize)]
struct Output {
    text: String,
    tooltip: String,
    class: &'static str,
}

fn request_key() -> Result<String, String> {
    let mut conn =
        UnixStream::connect(DAEMON_SOCKET).map_err(|e| format!("connect to daemon: {e}"))?;
    conn.write_all(format!("{DAEMON_KEY}\n").as_bytes())
        .map_err(|e| format!("write request: {e}"))?;

    let mut reader = BufReader::new(&mut conn);
    let mut line = String::new();
    reader
        .read_line(&mut line)
        .map_err(|e| format!("read status: {e}"))?;
    if !line.starts_with("OK") {
        return Err(format!("daemon: {}", line.trim()));
    }
    let mut body = String::new();
    reader
        .read_to_string(&mut body)
        .map_err(|e| format!("read body: {e}"))?;
    Ok(body)
}

fn fetch_status() -> Result<StatusResponse, String> {
    let body = request_key()?;
    serde_json::from_str(&body).map_err(|e| format!("parse status: {e}"))
}

fn handle_socket_error(e: String) {
    let out = Output {
        text: String::from("FATAL ERROR PARSING RESPONSE FROM UNIX SOCKET"),
        class: "unhealthy",
        tooltip: e,
    };
    println!("{}", serde_json::to_string(&out).unwrap());
    std::process::exit(1);
}

fn form_tooltip(data: &StatusResponse) -> String {
    let mut tooltip = format!("Totals: {:.2} hours", data.totals);
    tooltip += &format!("\nHours today: {:.2}", data.today_hours);
    tooltip += if data.quota_met {
        "\nQuota: met"
    } else {
        "\nQuota: not met"
    };
    tooltip += if data.break_met {
        "\nBreak: met"
    } else {
        "\nBreak: not met"
    };
    return tooltip;
}

fn form_class(data: &StatusResponse) -> &'static str {
    let class = if data.quota_met && data.break_met {
        "fully-met"
    } else if data.quota_met {
        "quota-met"
    } else if data.break_met {
        "break-met"
    } else {
        "quota-unmet"
    };
    return class;
}

fn render(data: &StatusResponse) -> Output {
    Output {
        text: format!("{:.2}", data.totals),
        tooltip: form_tooltip(data),
        class: form_class(data),
    }
}

fn main() {
    match fetch_status() {
        Ok(s) => println!("{}", serde_json::to_string(&render(&s)).unwrap()),
        Err(e) => handle_socket_error(e),
    }
}
