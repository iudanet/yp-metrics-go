package agent

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

func (a *Agent) PollWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.PollInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			a.logger.Info("PollWorker: context canceled, stopping")
			return
		case <-ticker.C:
			a.GetMetrics(ctx)

		}
	}

}
func (a *Agent) ReportWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.ReportInterval) * time.Second)
	defer ticker.Stop()
	for {

		select {
		case <-ctx.Done():
			a.logger.Info("ReportWorker: context canceled, stopping")
			return
		case <-ticker.C:
			counter, err := a.reader.GetMapCounter(ctx)
			if err != nil {
				a.logger.Error("Ошибка получения счетчика:", zap.Error(err))
				continue
			}
			for nameCounter, valueCounter := range counter {
				err = a.PushCounter(nameCounter, valueCounter)
				if err != nil {
					a.logger.Error("Ошибка отправки счетчика:", zap.Error(err))
					continue
				}
			}
			gaugeMap, err := a.reader.GetMapGauge(ctx)
			if err != nil {
				a.logger.Error("Ошибка получения гауга:", zap.Error(err))
				continue
			}
			for nameGauge, valueGauge := range gaugeMap {
				err = a.PushGauge(nameGauge, valueGauge)
				if err != nil {
					a.logger.Error("Ошибка отправки гауга:", zap.Error(err))
					continue
				}
			}
		}
	}
}

func (a *Agent) ReportWorkerBatch(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.ReportInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("ReportWorkerBatch: context canceled, stopping")
			close(a.metricsCh) // Закрываем канал для завершения воркеров
			a.workerWg.Wait()  // Ждем завершения всех воркеров
			return
		case <-ticker.C:
			metrics, err := a.getMetrics(ctx)
			if err != nil {
				a.logger.Error("Failed to get metrics", zap.Error(err))
				continue
			}

			// Отправляем метрики в канал для обработки воркерами
			select {
			case a.metricsCh <- metrics:
				// Метрики успешно отправлены в канал
			default:
				a.logger.Warn("Metrics channel is full, dropping batch")
			}
		}
	}
}
func (a *Agent) StartWorkers(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < a.config.RateLimit; i++ {
		wg.Add(1)
		go a.worker(ctx, wg)
	}
}

func (a *Agent) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for metrics := range a.metricsCh {
		if len(metrics) > 0 {
			err := a.PushMetricsBatch(metrics)
			if err != nil {
				a.logger.Error("Failed to push metrics batch", zap.Error(err))
			}
		}
	}

}
