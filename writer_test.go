package gotel

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicWriter_WriteHeader(t *testing.T) {
	t.Run("writes header once", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w}

		bw.WriteHeader(http.StatusOK)
		if bw.Status() != http.StatusOK {
			t.Errorf("expected status 200, got %d", bw.Status())
		}

		// Second call should be ignored
		bw.WriteHeader(http.StatusBadRequest)
		if bw.Status() != http.StatusOK {
			t.Errorf("expected status to remain 200, got %d", bw.Status())
		}
	})

	t.Run("handles informational status codes", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w}

		// 1xx status codes (except 101) should not set wroteHeader
		bw.WriteHeader(http.StatusContinue) // 100
		if bw.wroteHeader {
			t.Error("wroteHeader should be false for 100 Continue")
		}

		bw.WriteHeader(http.StatusOK)
		if bw.Status() != http.StatusOK {
			t.Errorf("expected status 200, got %d", bw.Status())
		}
	})

	t.Run("handles switching protocols", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w}

		bw.WriteHeader(http.StatusSwitchingProtocols) // 101
		if !bw.wroteHeader {
			t.Error("wroteHeader should be true for 101 Switching Protocols")
		}
	})

	t.Run("respects discard flag", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w, discard: true}

		bw.WriteHeader(http.StatusOK)
		// When discard is true, WriteHeader still writes to the underlying ResponseWriter
		// but Write() will not write the body
		if bw.Status() != http.StatusOK {
			t.Errorf("expected status 200, got %d", bw.Status())
		}
	})
}

func TestBasicWriter_Write(t *testing.T) {
	t.Run("writes data and tracks bytes", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w}

		data := []byte("test data")
		n, err := bw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}

		if bw.BytesWritten() != len(data) {
			t.Errorf("expected BytesWritten %d, got %d", len(data), bw.BytesWritten())
		}

		if w.Body.String() != string(data) {
			t.Errorf("expected body '%s', got '%s'", string(data), w.Body.String())
		}
	})

	t.Run("writes to tee writer", func(t *testing.T) {
		w := httptest.NewRecorder()
		teeBuffer := &bytes.Buffer{}
		bw := &basicWriter{ResponseWriter: w}
		bw.Tee(teeBuffer)

		data := []byte("test data")
		n, err := bw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}

		// Check both the response writer and tee buffer received the data
		if w.Body.String() != string(data) {
			t.Errorf("expected response body '%s', got '%s'", string(data), w.Body.String())
		}

		if teeBuffer.String() != string(data) {
			t.Errorf("expected tee buffer '%s', got '%s'", string(data), teeBuffer.String())
		}
	})

	t.Run("discards writes to response writer but writes to tee", func(t *testing.T) {
		w := httptest.NewRecorder()
		teeBuffer := &bytes.Buffer{}
		bw := &basicWriter{ResponseWriter: w}
		bw.Tee(teeBuffer)
		bw.Discard()

		data := []byte("test data")
		n, err := bw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}

		// Response writer should be empty
		if w.Body.Len() != 0 {
			t.Errorf("expected response body to be empty, got '%s'", w.Body.String())
		}

		// Tee buffer should have the data
		if teeBuffer.String() != string(data) {
			t.Errorf("expected tee buffer '%s', got '%s'", string(data), teeBuffer.String())
		}
	})

	t.Run("discards writes when no tee writer", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w}
		bw.Discard()

		data := []byte("test data")
		n, err := bw.Write(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}

		// Response writer should be empty
		if w.Body.Len() != 0 {
			t.Errorf("expected response body to be empty, got '%s'", w.Body.String())
		}

		// BytesWritten should still track the bytes
		if bw.BytesWritten() != len(data) {
			t.Errorf("expected BytesWritten %d, got %d", len(data), bw.BytesWritten())
		}
	})

	t.Run("automatically writes header on first write", func(t *testing.T) {
		w := httptest.NewRecorder()
		bw := &basicWriter{ResponseWriter: w}

		data := []byte("test")
		bw.Write(data)

		if bw.Status() != http.StatusOK {
			t.Errorf("expected status 200, got %d", bw.Status())
		}

		if !bw.wroteHeader {
			t.Error("expected wroteHeader to be true")
		}
	})
}

func TestBasicWriter_Unwrap(t *testing.T) {
	w := httptest.NewRecorder()
	bw := &basicWriter{ResponseWriter: w}

	unwrapped := bw.Unwrap()
	if unwrapped != w {
		t.Error("Unwrap should return the original ResponseWriter")
	}
}
