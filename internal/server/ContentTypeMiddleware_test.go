package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckContentType(t *testing.T) {
	// Создаем mock сервиса
	s := &service{}

	// Создаем тестовый обработчик, который просто возвращает 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Valid JSON content type",
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Empty content type",
			contentType:    "",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Invalid content type",
			contentType:    "text/plain",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Content type with charset",
			contentType:    "application/json; charset=utf-8",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Content type with version",
			contentType:    "application/json;version=1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Content type with spaces",
			contentType:    "application/json ; charset=utf-8",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем запрос с указанным Content-Type
			req := httptest.NewRequest("POST", "http://example.com", nil)
			req.Header.Set("Content-Type", tt.contentType)

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем middleware с тестовым обработчиком
			handler := s.CheckContentType(nextHandler)
			handler.ServeHTTP(rr, req)

			// Проверяем код статуса
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}
