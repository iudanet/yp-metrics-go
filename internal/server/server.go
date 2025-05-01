package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/iudanet/yp-metrics-go/internal/config"
	"github.com/iudanet/yp-metrics-go/internal/storage"
)

func NewService(storage storage.Repository, cfg *config.ServerConfig) *Service {
	return &Service{storage: storage, config: cfg}
}

type Service struct {
	storage storage.Repository
	config  *config.ServerConfig
}
type IndexData struct {
	Counters map[string]int64
	Gauges   map[string]float64
}

func (s *Service) UpdateMetric(w http.ResponseWriter, req *http.Request) {
	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")
	rawValue := req.PathValue("value")
	log.Printf("Received metric: type=%s name=%s value=%s", typeMetrics, name, rawValue)
	switch typeMetrics {
	case "gauge":
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
		err = s.storage.SetGauge(name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "counter":
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
		err = s.storage.SetCounter(name, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (s *Service) GetMetric(w http.ResponseWriter, req *http.Request) {
	typeMetrics := req.PathValue("typeMetrics")
	name := req.PathValue("name")

	switch typeMetrics {
	case "gauge":
		value, err := s.storage.GetGauge(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		fmt.Fprint(w, strconv.FormatFloat(value, 'f', -1, 64))
	case "counter":
		value, err := s.storage.GetCounter(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, "%d\n", value)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

func (s *Service) GetIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
	counters, err := s.storage.GetMapCounter()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	gauges, err := s.storage.GetMapGauge()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := IndexData{
		Counters: counters,
		Gauges:   gauges,
	}

	// Парсим HTML-шаблон
	tmpl := template.Must(template.New("index").Parse(`
  <!DOCTYPE html>
  <html lang="en">
  <head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>Метрики</title>
  </head>
  <body>
   <h1>Метрики Counters</h1>
   <ul>
   {{range $key, $value := .Counters}}
    <li>{{$key}}: {{$value}}</li>
   {{end}}
   </ul>
   <h1>Метрики Gauges</h1>
   <ul>
   {{range $key, $value := .Gauges}}
    <li>{{$key}}: {{printf "%4.3f" $value}}</li>
   {{end}}
   </ul>
  </body>
  </html>
 `))

	// Рендерим шаблон
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
