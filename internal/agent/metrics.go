package agent

import (
	"context"
	"fmt"
	"runtime"

	"github.com/iudanet/yp-metrics-go/internal/models"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
)

func (a *Agent) getMemStats(ctx context.Context) {
	runtime.ReadMemStats(a.memstats)
	a.memStatsMapper(ctx)
}

func (a *Agent) memStatsMapper(ctx context.Context) error {
	if err := a.writer.SetGauge(ctx, "Alloc", float64(a.memstats.Alloc)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "BuckHashSys", float64(a.memstats.BuckHashSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "Frees", float64(a.memstats.Frees)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "GCCPUFraction", a.memstats.GCCPUFraction); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "GCSys", float64(a.memstats.GCSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "HeapAlloc", float64(a.memstats.HeapAlloc)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "HeapIdle", float64(a.memstats.HeapIdle)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "HeapInuse", float64(a.memstats.HeapInuse)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "HeapObjects", float64(a.memstats.HeapObjects)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "HeapReleased", float64(a.memstats.HeapReleased)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "HeapSys", float64(a.memstats.HeapSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "LastGC", float64(a.memstats.LastGC)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "Lookups", float64(a.memstats.Lookups)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "MCacheInuse", float64(a.memstats.MCacheInuse)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "MCacheSys", float64(a.memstats.MCacheSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "Mallocs", float64(a.memstats.Mallocs)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "MSpanInuse", float64(a.memstats.MSpanInuse)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "MSpanSys", float64(a.memstats.MSpanSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "NextGC", float64(a.memstats.NextGC)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "NumForcedGC", float64(a.memstats.NumForcedGC)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "NumGC", float64(a.memstats.NumGC)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "OtherSys", float64(a.memstats.OtherSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "PauseTotalNs", float64(a.memstats.PauseTotalNs)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "StackInuse", float64(a.memstats.StackInuse)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "StackSys", float64(a.memstats.StackSys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "Sys", float64(a.memstats.Sys)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "TotalAlloc", float64(a.memstats.TotalAlloc)); err != nil {
		return err
	}
	return nil
}

func (a *Agent) collectPSUtilMetrics(ctx context.Context) error {
	// Собираем метрики памяти
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get memory stats: %w", err)
	}

	if err := a.writer.SetGauge(ctx, "TotalMemory", float64(v.Total)); err != nil {
		return err
	}
	if err := a.writer.SetGauge(ctx, "FreeMemory", float64(v.Free)); err != nil {
		return err
	}

	// Собираем метрики CPU
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU stats: %w", err)
	}

	for i, p := range percent {
		if err := a.writer.SetGauge(ctx, fmt.Sprintf("CPUutilization%d", i+1), p); err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) getMetrics(ctx context.Context) ([]models.Metrics, error) {
	var metrics []models.Metrics

	// Добавляем счетчики
	counters, err := a.reader.GetMapCounter(ctx)
	if err != nil {
		a.logger.Error("Ошибка получения счетчиков:", zap.Error(err))
		return nil, err
	}
	for name, value := range counters {
		delta := value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		})
	}

	// Добавляем gauge метрики
	gauges, err := a.reader.GetMapGauge(ctx)
	if err != nil {
		a.logger.Error("Ошибка получения gauge метрик:", zap.Error(err))
		return nil, err
	}
	for name, value := range gauges {
		val := value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		})
	}
	return metrics, nil
}
