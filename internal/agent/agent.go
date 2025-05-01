package agent

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/storage"
	"github.com/iudanet/yp-metrics-go/internal/utils"
)

type Agent struct {
	memstats *runtime.MemStats
	config   *config.AgentConfig
	// storage  storage.Repository
	writer  storage.MetricWriter
	counter storage.CounterIncrementer
	reader  storage.MetricReader
}

func NewAgent(cfg *config.AgentConfig, storage storage.Repository) *Agent {
	agent := &Agent{
		memstats: &runtime.MemStats{},
		config:   cfg,
		writer:   storage,
		counter:  storage,
		reader:   storage,
	}
	return agent
}

func (a *Agent) GetMetrics() {
	// увеличиваем каждую иттерацию
	a.counter.IncrCounter("PollCount")
	// получаем статистику памяти
	a.getMemStats()
	// получаем рандомное число
	a.writer.SetGauge("RandomValue", utils.GetRandomNumber())
}

func (a *Agent) getMemStats() {
	runtime.ReadMemStats(a.memstats)
	a.memStatsMapper()
}

func (a *Agent) memStatsMapper() {
	a.writer.SetGauge("Alloc", float64(a.memstats.Alloc))
	a.writer.SetGauge("BuckHashSys", float64(a.memstats.BuckHashSys))
	a.writer.SetGauge("Frees", float64(a.memstats.Frees))
	a.writer.SetGauge("GCCPUFraction", a.memstats.GCCPUFraction)
	a.writer.SetGauge("GCSys", float64(a.memstats.GCSys))
	a.writer.SetGauge("HeapAlloc", float64(a.memstats.HeapAlloc))
	a.writer.SetGauge("HeapIdle", float64(a.memstats.HeapIdle))
	a.writer.SetGauge("HeapInuse", float64(a.memstats.HeapInuse))
	a.writer.SetGauge("HeapObjects", float64(a.memstats.HeapObjects))
	a.writer.SetGauge("HeapReleased", float64(a.memstats.HeapReleased))
	a.writer.SetGauge("HeapSys", float64(a.memstats.HeapSys))
	a.writer.SetGauge("LastGC", float64(a.memstats.LastGC))
	a.writer.SetGauge("Lookups", float64(a.memstats.Lookups))
	a.writer.SetGauge("MCacheInuse", float64(a.memstats.MCacheInuse))
	a.writer.SetGauge("MCacheSys", float64(a.memstats.MCacheSys))
	a.writer.SetGauge("Mallocs", float64(a.memstats.Mallocs))
	a.writer.SetGauge("MSpanInuse", float64(a.memstats.MSpanInuse))
	a.writer.SetGauge("MSpanSys", float64(a.memstats.MSpanSys))
	a.writer.SetGauge("NextGC", float64(a.memstats.NextGC))
	a.writer.SetGauge("NumForcedGC", float64(a.memstats.NumForcedGC))
	a.writer.SetGauge("NumGC", float64(a.memstats.NumGC))
	a.writer.SetGauge("OtherSys", float64(a.memstats.OtherSys))
	a.writer.SetGauge("PauseTotalNs", float64(a.memstats.PauseTotalNs))
	a.writer.SetGauge("StackInuse", float64(a.memstats.StackInuse))
	a.writer.SetGauge("StackSys", float64(a.memstats.StackSys))
	a.writer.SetGauge("Sys", float64(a.memstats.Sys))
	a.writer.SetGauge("TotalAlloc", float64(a.memstats.TotalAlloc))
}

func (a *Agent) PollWorker() {
	for {
		a.GetMetrics()
		time.Sleep(time.Duration(a.config.PollInterval))

	}

}
func (a *Agent) ReportWorker() {
	for {
		counter, err := a.reader.GetMapCounter()
		if err != nil {
			log.Println("Ошибка получения счетчика:", err)
			continue
		}
		for nameCouner, valueCounter := range counter {
			err = a.PushCounter(nameCouner, valueCounter)
			if err != nil {
				log.Println(err)
				continue
			}

		}
		gaugeMap, err := a.reader.GetMapGauge()
		if err != nil {
			log.Println("Ошибка получения счетчика:", err)
			continue
		}
		for nameGauge, valueGauge := range gaugeMap {
			err = a.PushGauge(nameGauge, valueGauge)
			if err != nil {
				log.Println(err)
				continue
			}

		}
		time.Sleep(time.Duration(a.config.ReportInterval) * time.Second)
	}

}
func (a *Agent) PushCounter(name string, value int64) error {
	//	POST /update/counter/someMetric/527 HTTP/1.1
	//
	// Host: localhost:8080
	// Content-Length: 0
	// Content-Type: text/plain
	req, err := http.Post(fmt.Sprintf("http://%s/update/%s/%s/%d", a.config.MetricServerHost, "counter", name, value), "text/plain", nil)
	if err != nil {
		return fmt.Errorf("unable to send request to server: %w", err)
	}
	defer req.Body.Close()
	if req.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push counter metric: %s", req.Status)
	}
	return nil
}

func (a *Agent) PushGauge(name string, value float64) error {
	//	POST /update/gauge/someMetric/527 HTTP/1.1
	//
	// Host: localhost:8080
	// Content-Length: 0
	// Content-Type: text/plain
	req, err := http.Post(fmt.Sprintf("http://%s/update/%s/%s/%f", a.config.MetricServerHost, "gauge", name, value), "text/plain", nil)
	if err != nil {
		return fmt.Errorf("unable to send request to server: %w", err)
	}
	defer req.Body.Close()
	if req.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push counter metric: %s", req.Status)
	}
	return nil
}
