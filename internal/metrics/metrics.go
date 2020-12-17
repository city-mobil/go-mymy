package metrics

import "github.com/prometheus/client_golang/prometheus"

type ReplState int8

const (
	StateStopped ReplState = iota
	StateDumping
	StateRunning
)

var (
	secondsBehindMaster = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mymy",
		Name:      "seconds_behind",
		Help:      "Current replication lag of the replicator. Calculates as diff between current timestamp and last event timestamp. The value updates only after receiving the event.",
	})

	syncedSecondsAgo = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mymy",
		Name:      "synced_seconds_ago",
		Help:      "Seconds since the last event has been synced",
	})

	replState = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mymy",
		Name:      "state",
		Help:      "The replication running state: 0=stopped, 1=dumping, 2=running",
	})
)

func Init() {
	prometheus.MustRegister(secondsBehindMaster)
	prometheus.MustRegister(replState)
	prometheus.MustRegister(syncedSecondsAgo)
}

func SetSecondsBehindMaster(value uint32) {
	secondsBehindMaster.Set(float64(value))
}

func SetSyncedSecondsAgo(sec int64) {
	syncedSecondsAgo.Set(float64(sec))
}

func SetReplicationState(state ReplState) {
	replState.Set(float64(state))
}
