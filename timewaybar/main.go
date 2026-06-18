package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	defaultAPI       = "http://time.lucia.friedmicro-lab.org"
	defaultStateFile = "/tmp/timestate.json"
	configFile       = "timewaybar.conf"
)

var (
	apiBase   = defaultAPI
	stateFile = defaultStateFile
)

func loadConfig() {
	f, err := os.Open(configFile)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		switch key {
		case "url", "api_url":
			if val != "" {
				apiBase = val
			}
		case "state_file":
			if val != "" {
				stateFile = val
			}
		}
	}
}

type Day struct {
	Date  string  `json:"date"`
	Hours float64 `json:"hours"`
}

type Response struct {
	BreakMet   bool           `json:"break_met"`
	Days       map[string]Day `json:"days"`
	QuotaMet   bool           `json:"quota_met"`
	TodayHours float64        `json:"today_hours"`
	Totals     float64        `json:"totals"`
}

type Output struct {
	Text       string  `json:"text"`
	Alt        float64 `json:"alt,omitempty"`
	Tooltip    string  `json:"tooltip"`
	Class      string  `json:"class"`
	Percentage float64 `json:"percentage"`
}

func main() {
	loadConfig()

	var data Response
	resp, err := http.Get(apiBase + "/api/week_times")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		data = loadState()
	} else {
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			fmt.Fprintf(os.Stderr, "error decoding JSON: %v\n", err)
			data = loadState()
		} else {
			saveState(data)
		}
	}

	tooltip := fmt.Sprintf("Totals: %.2f hours", data.Totals)
	tooltip += fmt.Sprintf("\nHours today: %.2f", data.TodayHours)
	if data.QuotaMet {
		tooltip += "\nQuota: met"
	} else {
		tooltip += "\nQuota: not met"
	}
	if data.BreakMet {
		tooltip += "\nBreak: met"
	} else {
		tooltip += "\nBreak: not met"
	}

	class := "quota-unmet"
	if data.QuotaMet && data.BreakMet {
		class = "fully-met"
	} else if data.QuotaMet {
		class = "quota-met"
	} else if data.BreakMet {
		class = "break-met"
	}

	out := Output{
		Text:       fmt.Sprintf("%.2f", data.Totals),
		Alt:        data.TodayHours,
		Tooltip:    tooltip,
		Class:      class,
		Percentage: 100,
	}

	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}

func saveState(data Response) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	os.WriteFile(stateFile, b, 0644)
}

func loadState() Response {
	var data Response
	b, err := os.ReadFile(stateFile)
	if err != nil {
		return data
	}
	json.Unmarshal(b, &data)
	return data
}
