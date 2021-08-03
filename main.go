package main

import (
	"net/http"
	"os"

	"github.com/coreos/go-iptables/iptables"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	MangleTable     = "mangle"
	PreroutingChain = "PREROUTING"
)

func main() {
	var (
		listenAddress = kingpin.Flag(
			"web.listen-address",
			"Address on which to expose metrics and web interface.",
		).Default(":9200").String()
	)
	logger := log.NewLogfmtLogger(os.Stdout)

	c, err := newFwMarkCollector()
	if err != nil {
		level.Error(logger).Log("msg", "failed to start collector")
		os.Exit(1)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	if err := http.ListenAndServe(*listenAddress, promhttp.HandlerFor(reg, promhttp.HandlerOpts{})); err != nil {
		level.Error(logger).Log("msg", "failed to start http server")
		os.Exit(1)
	}
}

type fwMarkCollector struct {
	t                *iptables.IPTables
	duplicateFWMark  *prometheus.Desc
	rulesListSuccess *prometheus.Desc
}

func newFwMarkCollector() (*fwMarkCollector, error) {
	t, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	return &fwMarkCollector{
		t: t,
		duplicateFWMark: prometheus.NewDesc(
			"fwmark_duplication",
			"Whether there is any duplicate fwmark or not. 1 represents duplication and 0 represents no duplication.",
			nil, nil,
		),
		rulesListSuccess: prometheus.NewDesc(
			"fwmark_rules_list_success",
			"Whether iptables rules list succeeded or not. 1 represents success and 0 represents failure.",
			nil, nil,
		),
	}, err
}

// Describe returns all descriptions of the collector.
func (c *fwMarkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.duplicateFWMark
}

func (c *fwMarkCollector) Collect(ch chan<- prometheus.Metric) {
	// Add rules.
	_, err := c.t.List(MangleTable, PreroutingChain)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.rulesListSuccess, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.rulesListSuccess, prometheus.GaugeValue, 1)
}
