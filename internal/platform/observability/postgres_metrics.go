package observability

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type pgxPoolCollector struct {
	pool *pgxpool.Pool
	desc map[string]*prometheus.Desc
}

func RegisterPostgresPoolMetrics(pool *pgxpool.Pool) error {
	if pool == nil {
		return nil
	}

	collector := &pgxPoolCollector{
		pool: pool,
		desc: map[string]*prometheus.Desc{
			"acquired_connections": prometheus.NewDesc("app_postgres_pool_acquired_connections", "Current number of acquired PostgreSQL connections.", nil, nil),
			"idle_connections":     prometheus.NewDesc("app_postgres_pool_idle_connections", "Current number of idle PostgreSQL connections.", nil, nil),
			"total_connections":    prometheus.NewDesc("app_postgres_pool_total_connections", "Current number of total PostgreSQL connections.", nil, nil),
			"max_connections":      prometheus.NewDesc("app_postgres_pool_max_connections", "Configured max PostgreSQL connections in pool.", nil, nil),
			"constructing_connections": prometheus.NewDesc("app_postgres_pool_constructing_connections", "Current number of PostgreSQL connections being constructed.", nil, nil),
			"acquire_total":            prometheus.NewDesc("app_postgres_pool_acquire_total", "Total successful PostgreSQL connection acquires.", nil, nil),
			"acquire_canceled_total":   prometheus.NewDesc("app_postgres_pool_acquire_canceled_total", "Total canceled PostgreSQL connection acquires.", nil, nil),
			"empty_acquire_total":      prometheus.NewDesc("app_postgres_pool_empty_acquire_total", "Total acquires when pool had no idle connections.", nil, nil),
			"new_connections_total":    prometheus.NewDesc("app_postgres_pool_new_connections_total", "Total new PostgreSQL connections created by pool.", nil, nil),
			"max_idle_destroy_total":   prometheus.NewDesc("app_postgres_pool_max_idle_destroy_total", "Total PostgreSQL connections destroyed due to max idle limit.", nil, nil),
			"max_lifetime_destroy_total": prometheus.NewDesc("app_postgres_pool_max_lifetime_destroy_total", "Total PostgreSQL connections destroyed due to max lifetime.", nil, nil),
			"acquire_duration_seconds_total": prometheus.NewDesc("app_postgres_pool_acquire_duration_seconds_total", "Total duration spent acquiring PostgreSQL connections in seconds.", nil, nil),
			"empty_acquire_wait_seconds_total": prometheus.NewDesc("app_postgres_pool_empty_acquire_wait_seconds_total", "Total wait time for acquires from an empty PostgreSQL pool in seconds.", nil, nil),
		},
	}

	if err := prometheus.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return nil
		}
		return err
	}

	return nil
}

func (c *pgxPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.desc {
		ch <- desc
	}
}

func (c *pgxPoolCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.pool.Stat()

	ch <- prometheus.MustNewConstMetric(c.desc["acquired_connections"], prometheus.GaugeValue, float64(stats.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.desc["idle_connections"], prometheus.GaugeValue, float64(stats.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.desc["total_connections"], prometheus.GaugeValue, float64(stats.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.desc["max_connections"], prometheus.GaugeValue, float64(stats.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.desc["constructing_connections"], prometheus.GaugeValue, float64(stats.ConstructingConns()))

	ch <- prometheus.MustNewConstMetric(c.desc["acquire_total"], prometheus.CounterValue, float64(stats.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.desc["acquire_canceled_total"], prometheus.CounterValue, float64(stats.CanceledAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.desc["empty_acquire_total"], prometheus.CounterValue, float64(stats.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.desc["new_connections_total"], prometheus.CounterValue, float64(stats.NewConnsCount()))
	ch <- prometheus.MustNewConstMetric(c.desc["max_idle_destroy_total"], prometheus.CounterValue, float64(stats.MaxIdleDestroyCount()))
	ch <- prometheus.MustNewConstMetric(c.desc["max_lifetime_destroy_total"], prometheus.CounterValue, float64(stats.MaxLifetimeDestroyCount()))
	ch <- prometheus.MustNewConstMetric(c.desc["acquire_duration_seconds_total"], prometheus.CounterValue, stats.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(c.desc["empty_acquire_wait_seconds_total"], prometheus.CounterValue, stats.EmptyAcquireWaitTime().Seconds())
}
