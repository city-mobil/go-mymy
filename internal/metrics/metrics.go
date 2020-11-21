package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	secondsBehindMaster = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mymy",
		Name:      "seconds_behind",
		Help:      "Current replication lag of the replicator",
	})

	replState = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mymy",
		Name:      "state",
		Help:      "The replication running state: 0=stopped, 1=ok",
	})
)

func Init() {
	prometheus.MustRegister(secondsBehindMaster)
	prometheus.MustRegister(replState)
}

func SetSecondsBehindMaster(value uint32) {
	secondsBehindMaster.Set(float64(value))
}

func SetReplicationState(state bool) {
	v := 0
	if state {
		v = 1
	}
	replState.Set(float64(v))
}
