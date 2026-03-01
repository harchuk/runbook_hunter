package obs

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	SignalsIngested prometheus.Counter
	IncidentsOpen   prometheus.Gauge
	StepRunsTotal   *prometheus.CounterVec
	TelegramSent    prometheus.Counter
	MattermostSent  prometheus.Counter
	ErrorsTotal     *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		SignalsIngested: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "signals_ingested_total",
			Help: "Total ingested signals from Alertmanager",
		}),
		IncidentsOpen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "incidents_open",
			Help: "Open incidents count",
		}),
		StepRunsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "step_runs_total",
			Help: "Total runbook step runs",
		}, []string{"status"}),
		TelegramSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "telegram_sent_total",
			Help: "Total sent Telegram updates",
		}),
		MattermostSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mattermost_sent_total",
			Help: "Total sent Mattermost updates",
		}),
		ErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Errors by component",
		}, []string{"component"}),
	}

	reg.MustRegister(
		m.SignalsIngested,
		m.IncidentsOpen,
		m.StepRunsTotal,
		m.TelegramSent,
		m.MattermostSent,
		m.ErrorsTotal,
	)
	return m
}
