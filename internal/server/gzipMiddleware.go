package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// CompressibleContentTypes maps MIME types that should be compressed
var compressibleContentTypes = map[string]bool{
	"application/javascript": true,
	"application/json":       true,
	"text/css":               true,
	"text/html":              true,
	"text/plain":             true,
	"text/xml":               true,
}

// compressWriter implements http.ResponseWriter with transparent gzip compression
// and proper HTTP header handling
type compressWriter struct {
	w            http.ResponseWriter
	zw           *gzip.Writer
	headerWritten bool
	shouldCompress bool
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:            w,
		zw:           gzip.NewWriter(w),
		headerWritten: false,
		shouldCompress: false,
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.headerWritten {
		// Check content type before writing
		contentType := c.w.Header().Get("Content-Type")
		c.shouldCompress = isCompressibleType(contentType)
		
		if c.shouldCompress {
			c.w.Header().Set("Content-Encoding", "gzip")
		}
		c.headerWritten = true
	}
	
	if c.shouldCompress {
		return c.zw.Write(p)
	}
	
	return c.w.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if !c.headerWritten {
		// Check content type before writing header
		contentType := c.w.Header().Get("Content-Type")
		c.shouldCompress = isCompressibleType(contentType) && statusCode < 300
		
		if c.shouldCompress {
			c.w.Header().Set("Content-Encoding", "gzip")
		}
		c.headerWritten = true
	}
	
	c.w.WriteHeader(statusCode)
}

// Close flushes and closes the gzip.Writer
func (c *compressWriter) Close() error {
	if c.shouldCompress {
		return c.zw.Close()
	}
	return nil
}

// isCompressibleType checks if the given content type should be compressed
func isCompressibleType(contentType string) bool {
	// Extract the base content type without parameters
	if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	
	return compressibleContentTypes[contentType]
}

// compressReader implements io.ReadCloser for transparent gzip decompression
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}
func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	// Close both readers and return any error that occurs
	rErr := c.r.Close()
	zrErr := c.zr.Close()
	
	// Return the first error encountered
	if rErr != nil {
		return rErr
	}
	return zrErr
}

// GzipMiddleware wraps an http.Handler with gzip compression and decompression
func (s *service) GzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip encoding
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		
		var responseWriter http.ResponseWriter = w
		
		if supportsGzip {
			// Only create compressWriter if client supports gzip
			cw := newCompressWriter(w)
			responseWriter = cw
			defer cw.Close()
		}
		
		// Check if request body is gzip encoded
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			
			r.Body = cr
			defer cr.Close()
		}
		
		// Pass control to the wrapped handler
		h.ServeHTTP(responseWriter, r)
	})
}