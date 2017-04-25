package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHappyPath(t *testing.T) {
	var fixtures = []fixture{
		{
			in: metrics{
				{
					Metric: "tsd.compaction.duplicates",
					Value:  "1",
					Tags:   map[string]string{"host": "foo", "type": "variant1"},
				},
				{
					Metric: "tsd.compaction.duplicates",
					Value:  "2",
					Tags:   map[string]string{"host": "bar", "type": "variant2"},
				},
			},
			out: ``,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	e := newExporter(ts.URL, 5*time.Second)
	ch := make(chan prometheus.Metric)
	go e.Collect(ch)
	m := <-ch
	for _, f := range fixtures {
		t.Fatalf(m.Desc().String(), f.out)
	}
}

type fixture struct {
	in  metrics
	out string
}
