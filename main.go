package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const (
	applicationName = "opentsdb_exporter"
	metricsRoute    = "/metrics"
	probeRoute      = "/opentsdb"
)

var (
	nameSanitiser = strings.NewReplacer("tsd.", "opentsdb_", ".", "_", "-", "_")
	revision      = "unknown"

	versionString = fmt.Sprintf("%s %s (%s)", applicationName, revision, runtime.Version())
	showVersion   = flag.Bool("version", false, "Print version information.")
	listenAddress = flag.String("web.listen-address", ":9250", "Address to listen on for web interface and telemetry.")
	timeout       = flag.Duration("opentsdb.timeout", 5*time.Second, "Timeout for HTTP requests to OpenTSDB.")
)

func main() {

	flag.Parse()

	if *showVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	log.Infoln("Starting", versionString)

	http.Handle(metricsRoute, promhttp.Handler())
	http.HandleFunc("/opentsdb", handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head><title>OpenTSDB Exporter</title></head>
		<body>
		<h1>` + versionString + `</h1>
		<p><a href='` + metricsRoute + `'>Metrics</a></p>
		<p><a href='` + probeRoute + `?target=http://opentsdb.localhost'>Example OpenTSDB probe</a></p>
		</body>
		</html>`))
	})

	log.Infoln("Listening on", *listenAddress)

	s := &http.Server{
		Addr:         *listenAddress,
		ReadTimeout:  *timeout * 2,
		WriteTimeout: *timeout * 2,
	}
	log.Fatal(s.ListenAndServe())
}

type collector struct {
	target  string
	timeout time.Duration
}

func handler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}

	log.Debugf("Scraping target %q", target)
	registry := prometheus.NewRegistry()
	registry.MustRegister(&collector{target: target, timeout: *timeout})

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	metrics, err := c.queryOpenTSDB()
	if err != nil {
		log.Errorf("Error scraping target %s: %s", c.target, err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("api_error", "Error scraping target", nil, nil), err)
		return
	}
	for _, m := range *metrics {
		value, err := m.Value.Float64()
		if err != nil {
			log.Errorf("Error scraping target %s: %s", c.target, err)
			ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("api_error", "Error scraping target", nil, nil), err)
			return
		}

		var labelNames, labelValues []string
		for k, v := range m.Tags {
			if k == "host" {
				continue
			}

			labelNames = append(labelNames, k)
			labelValues = append(labelValues, v)
		}

		name := nameSanitiser.Replace(m.Metric)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(name, "", labelNames, nil),
			prometheus.GaugeValue,
			value,
			labelValues...,
		)
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	// Copied from https://github.com/prometheus/snmp_exporter/blob/bc4b0a4db1e22c5379f7fdb3f314dbb58a48c637/collector.go#L118
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c *collector) queryOpenTSDB() (m *metrics, err error) {
	client := &http.Client{
		Timeout: c.timeout,
	}

	resp, err := client.Get(c.target + "/api/stats")
	if err != nil {
		return m, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(body, &m); err != nil {
		return m, err
	}

	return
}

type metrics []struct {
	Metric string `json:"metric"`
	// We don't care about the timestamp, let Prometheus use the scrape time
	// https://prometheus.io/docs/instrumenting/writing_exporters/#scheduling
	Value json.Number       `json:"value"`
	Tags  map[string]string `json:"tags"`
}
