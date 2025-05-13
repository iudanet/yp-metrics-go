package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/models"
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
		for nameCounter, valueCounter := range counter {
			err = a.PushCounter(nameCounter, valueCounter)
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
	metric := models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}
	
	return a.sendCompressedMetric(&metric)
}

func (a *Agent) PushGauge(name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}
	
	return a.sendCompressedMetric(&metric)
}

// compressData compresses data using gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}
	
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	
	return buf.Bytes(), nil
}

// sendCompressedMetric sends a metric in JSON format with gzip compression
func (a *Agent) sendCompressedMetric(metric *models.Metrics) error {
	// Convert metric to JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric to JSON: %w", err)
	}
	
	// Compress JSON data
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}
	
	// Create request
	url := fmt.Sprintf("http://%s/update/", a.config.MetricServerHost)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	
	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push metric: %s", resp.Status)
	}
	
	return nil
}
