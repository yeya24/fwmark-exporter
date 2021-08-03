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
	kingpin.Parse()
	logger := log.NewLogfmtLogger(os.Stdout)

	c, err := newFwMarkCollector(logger)
	if err != nil {
		level.Error(logger).Log("msg", "failed to start collector", "error", err)
		os.Exit(1)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	if err := http.ListenAndServe(*listenAddress, promhttp.HandlerFor(reg, promhttp.HandlerOpts{})); err != nil {
		level.Error(logger).Log("msg", "failed to start http server", "error", err)
		os.Exit(1)
	}
}

type fwMarkCollector struct {
	t                *iptables.IPTables
	duplicateFWMark  *prometheus.Desc
	rulesListSuccess *prometheus.Desc
	logger           log.Logger
}

func newFwMarkCollector(logger log.Logger) (*fwMarkCollector, error) {
	t, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	return &fwMarkCollector{
		t:      t,
		logger: logger,
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
	rules, err := c.t.Stats(MangleTable, PreroutingChain)
	if err != nil {
		level.Error(c.logger).Log("msg", "failed to list iptables rules", "error", err)
		ch <- prometheus.MustNewConstMetric(c.rulesListSuccess, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.rulesListSuccess, prometheus.GaugeValue, 1)

	set := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		s, err := c.t.ParseStat(rule)
		if err != nil {
			continue
		}
		if s.Target != "MARK" {
			continue
		}

		if _, ok := set[s.Options]; ok {
			ch <- prometheus.MustNewConstMetric(c.duplicateFWMark, prometheus.GaugeValue, 1)
			return
		} else {
			set[s.Options] = struct{}{}
		}
	}

	ch <- prometheus.MustNewConstMetric(c.duplicateFWMark, prometheus.GaugeValue, 0)
}
