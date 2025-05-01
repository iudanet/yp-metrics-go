package agent

import (
	"errors"
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
	storage  storage.Repository
}

func NewAgent(cfg *config.AgentConfig, storage storage.Repository) *Agent {
	agent := &Agent{
		memstats: &runtime.MemStats{},
		config:   cfg,
		storage:  storage,
	}
	return agent
}

func (a *Agent) GetMetrics() {
	// увеличиваем каждую иттерацию
	a.storage.IncrCounter("PollCount")
	// получаем статистику памяти
	a.getMemStats()
	// получаем рандомное число
	a.storage.SetGauge("RandomValue", utils.GetRandomNumber())
}

func (a *Agent) getMemStats() {
	runtime.ReadMemStats(a.memstats)
	a.memStatsMapper()
}

func (a *Agent) memStatsMapper() {
	a.storage.SetGauge("Alloc", float64(a.memstats.Alloc))
	a.storage.SetGauge("BuckHashSys", float64(a.memstats.BuckHashSys))
	a.storage.SetGauge("Frees", float64(a.memstats.Frees))
	a.storage.SetGauge("GCCPUFraction", a.memstats.GCCPUFraction)
	a.storage.SetGauge("GCSys", float64(a.memstats.GCSys))
	a.storage.SetGauge("HeapAlloc", float64(a.memstats.HeapAlloc))
	a.storage.SetGauge("HeapIdle", float64(a.memstats.HeapIdle))
	a.storage.SetGauge("HeapInuse", float64(a.memstats.HeapInuse))
	a.storage.SetGauge("HeapObjects", float64(a.memstats.HeapObjects))
	a.storage.SetGauge("HeapReleased", float64(a.memstats.HeapReleased))
	a.storage.SetGauge("HeapSys", float64(a.memstats.HeapSys))
	a.storage.SetGauge("LastGC", float64(a.memstats.LastGC))
	a.storage.SetGauge("Lookups", float64(a.memstats.Lookups))
	a.storage.SetGauge("MCacheInuse", float64(a.memstats.MCacheInuse))
	a.storage.SetGauge("MCacheSys", float64(a.memstats.MCacheSys))
	a.storage.SetGauge("Mallocs", float64(a.memstats.Mallocs))
	a.storage.SetGauge("MSpanInuse", float64(a.memstats.MSpanInuse))
	a.storage.SetGauge("MSpanSys", float64(a.memstats.MSpanSys))
	a.storage.SetGauge("NextGC", float64(a.memstats.NextGC))
	a.storage.SetGauge("NumForcedGC", float64(a.memstats.NumForcedGC))
	a.storage.SetGauge("NumGC", float64(a.memstats.NumGC))
	a.storage.SetGauge("OtherSys", float64(a.memstats.OtherSys))
	a.storage.SetGauge("PauseTotalNs", float64(a.memstats.PauseTotalNs))
	a.storage.SetGauge("StackInuse", float64(a.memstats.StackInuse))
	a.storage.SetGauge("StackSys", float64(a.memstats.StackSys))
	a.storage.SetGauge("Sys", float64(a.memstats.Sys))
	a.storage.SetGauge("TotalAlloc", float64(a.memstats.TotalAlloc))
}

func (a *Agent) PollWorker() {
	for {
		a.GetMetrics()
		time.Sleep(time.Duration(a.config.PollInterval))

	}

}
func (a *Agent) ReportWorker() {
	for {
		counter, err := a.storage.GetMapCounter()
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
		gaugeMap, err := a.storage.GetMapGauge()
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
		log.Println(err)
		return err
	}
	defer req.Body.Close()
	if req.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("failed to push counter metric: %s", req.Status))
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
		log.Println(err)
		return err
	}
	defer req.Body.Close()
	if req.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("failed to push counter metric: %s", req.Status))
	}
	return nil
}
