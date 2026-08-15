package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRejectsRequestBody(t *testing.T) {
	called := false
	handler := New(func(_ []string) { called = true }, nil)
	request := httptest.NewRequest(http.MethodPost, handler.Path, strings.NewReader("x"))
	response := httptest.NewRecorder()

	handler.Handle(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	if called {
		t.Fatal("update function was called for a request with a body")
	}
}

func TestHandleStopsWaitingWhenRequestIsCanceled(t *testing.T) {
	updateLock := make(chan bool)
	called := false
	handler := New(func(_ []string) { called = true }, updateLock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, handler.Path+"?image=example/image", nil).WithContext(ctx)

	handler.Handle(httptest.NewRecorder(), request)

	if called {
		t.Fatal("update function was called after the request was canceled")
	}
}
