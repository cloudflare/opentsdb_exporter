# OpenTSDB Exporter

A daemon that takes the URL of an [OpenTSDB][] server and exposes Prometheus
metrics based on data scraped from OpenTSDB's stats API.

See OpenTSDB's [API documentation on the stats
endpoint](http://opentsdb.net/docs/build/html/api_http/stats/index.html).

Tested on OpenTSDB 2.3.

[OpenTSDB]: http://opentsdb.net/

## Prerequisites

To use the OpenTSDB exporter, you'll need:

- an [OpenTSDB][] cluster

To build, you'll need:

- [Make][]
- [Go][] 1.8 or above
- a working [GOPATH][]

[Make]: https://www.gnu.org/software/make/
[Go]: https://golang.org/dl/
[GOPATH]: https://golang.org/cmd/go/#hdr-GOPATH_environment_variable

## Limitations

The metrics generated are translated one-to-one from the metrics exposed on the
stats endpoint, so do not follow the [Prometheus metric naming conventions][].
This is primarily to keep the code simple.

[Prometheus metric naming conventions]: https://prometheus.io/docs/practices/naming/

## How to run

    go get -u github.com/cloudflare/opentsdb_exporter
    cd $GOPATH/src/github.com/cloudflare/opentsdb_exporter
    make
    ./opentsdb_exporter

## How to run the tests

    make tests

## Contributions

Pull requests, comments and suggestions are welcome.

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more information.
