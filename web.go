package fetcher

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func StartWeb(config Config) {
	handleFunc("/", "GET", config, handleInfo)
	handleFunc("/log", "GET", config, handleLog)
	handleFunc("/status", "GET", config, handleStatus)

	handleFunc("/force/fetch", "POST", config, handleForceFetch)
	handleFunc("/force/sort", "POST", config, handleForceSort)

	addr := config.WebUI.Host

	log.Info("Server starting to listen on ", addr)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Error during serving web interface", err)
	}
}

func handleInfo(w http.ResponseWriter, r *http.Request, config Config) {
	fmt.Fprint(w, "API is online")
}

func handleStatus(w http.ResponseWriter, r *http.Request, config Config) {
	var res = make(map[string]interface{})

	res["running"] = true
	// locks are deprecated and do not matter in this implementation
	res["fetch_lock"] = false
	res["sort_lock"] = false

	sendJson(res, w)
}

func handleLog(w http.ResponseWriter, r *http.Request, config Config) {
	var res = make(map[string]interface{})

	res["success"] = true
	res["log"] = make([]string, 0)
	// HACK: this is a pretty dirty way to read the log
	out, err := exec.Command("journalctl", "--no-pager", "-u", "fetcherd", "--output=cat", "-n", "100").Output()
	if err != nil {
		log.WithField("err", err).Error("Failed to run journalctl command")
	} else {
		res["log"] = strings.Split(string(out), "\n")
	}

	sendJson(res, w)
}

func handleForceFetch(w http.ResponseWriter, r *http.Request, config Config) {
	var res = make(map[string]interface{})

	err := Fetch(config)
	res["success"] = err == nil
	if err != nil {
		res["error"] = fmt.Sprint(err)
	}

	sendJson(res, w)
}

func handleForceSort(w http.ResponseWriter, r *http.Request, config Config) {
	var res = make(map[string]interface{})

	// TODO: run sort command
	res["success"] = true

	sendJson(res, w)
}

func sendJson(d interface{}, w http.ResponseWriter) {
	bytes, err := json.Marshal(d)

	if err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.Write(bytes)
	} else {
		fmt.Fprint(w, err)
	}
}

func handleFunc(pattern string, method string, config Config, handler func(http.ResponseWriter, *http.Request, Config)) {

	// duplicates the handler with a trailing slash
	if pattern == strings.TrimRight(pattern, "/") {
		handleFunc(pattern+"/", method, config, handler)
	}

	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Trying Request for ", r.URL.Path, " (", r.Method, ") against ", pattern)
		matched := pattern == r.URL.Path
		matchedTrim := pattern == strings.TrimRight(r.URL.Path, "/")
		if method == r.Method && (matched || matchedTrim) {
			start := time.Now()
			handler(w, r, config)

			end := time.Now()
			log.WithField("path", r.URL.Path).WithField("method", r.Method).WithField("time", end.Sub(start)).Info("Request complete")
		} else {
			http.NotFound(w, r)
		}
	})
}
