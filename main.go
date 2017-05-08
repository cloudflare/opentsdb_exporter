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
	"github.com/prometheus/common/log"
)

const applicationName = "opentsdb_exporter"

var (
	nameSanitiser = strings.NewReplacer("tsd.", "opentsdb_", ".", "_", "-", "_")
	revision      = "unknown"
)

func main() {
	var (
		versionString = fmt.Sprintf("%s %s (%s)", applicationName, revision, runtime.Version())
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9250", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		url           = flag.String("opentsdb.url", "http://localhost:4242", "HTTP URL for OpenTSDB server to monitor.")
		timeout       = flag.Duration("opentsdb.timeout", 5*time.Second, "Timeout for HTTP requests to OpenTSDB.")
	)

	flag.Parse()

	if *showVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	log.Infoln("Starting", versionString)

	exporter := newExporter(*url, *timeout)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head><title>OpenTSDB Exporter</title></head>
		<body>
		<h1>` + versionString + `</h1>
		<p><a href='` + *metricsPath + `'>Metrics</a></p>
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

type exporter struct {
	url     string
	timeout time.Duration
}

func newExporter(url string, timeout time.Duration) *exporter {
	return &exporter{url, timeout}
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	metrics, err := e.queryOpenTSDB()
	if err != nil {
		log.Errorf("Error scraping target %s: %s", e.url, err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("api_error", "Error scraping target", nil, nil), err)
		return
	}
	for _, m := range *metrics {
		value, err := m.Value.Float64()
		if err != nil {
			log.Errorf("Error scraping target %s: %s", e.url, err)
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

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	// Copied from https://github.com/prometheus/snmp_exporter/blob/bc4b0a4db1e22c5379f7fdb3f314dbb58a48c637/collector.go#L118
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (e *exporter) queryOpenTSDB() (m *metrics, err error) {
	client := &http.Client{
		Timeout: e.timeout,
	}

	resp, err := client.Get(e.url + "/api/stats")
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
