package triemux

import "github.com/prometheus/client_golang/prometheus"

// TODO: don't use init for this.
func init() {
	registerMetrics(prometheus.DefaultRegisterer)
}
