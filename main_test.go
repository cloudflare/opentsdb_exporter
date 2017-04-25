package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHappyPath(t *testing.T) {
	var example = fixture{
		in: metrics{
			{
				Metric: "tsd.compaction.duplicates",
				Value:  "1",
				Tags:   map[string]string{"host": "foo", "type": "variant1"},
			},
			{
				Metric: "tsd.compaction.duplicates.test",
				Value:  "2",
				Tags:   map[string]string{"host": "bar", "type": "variant2"},
			},
		},
		out: []string{
			`Desc{fqName: "opentsdb_compaction_duplicates", help: "", constLabels: {}, variableLabels: [type]}`,
			`Desc{fqName: "opentsdb_compaction_duplicates_test", help: "", constLabels: {}, variableLabels: [type]}`,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := json.NewEncoder(w)
		_ = e.Encode(example.in)
	}))
	defer ts.Close()

	e := newExporter(ts.URL, 5*time.Second)
	ch := make(chan prometheus.Metric)
	go e.Collect(ch)

	var metricsFound []string
	for m := range ch {
		metricsFound = append(metricsFound, m.Desc().String())

		if len(metricsFound) == len(example.in) {
			break
		}
	}

	if !reflect.DeepEqual(metricsFound, example.out) {
		t.Fatalf("\n%s\n\tdoes not match expected output:\n%s",
			strings.Join(metricsFound, "\n"),
			strings.Join(example.out, "\n"),
		)
	}
}

type fixture struct {
	in  metrics
	out []string
}
