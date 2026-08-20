package routes

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestAddRoutesBodyDeadlineSurvivesProfileCustomisationHydrator(t *testing.T) {
	deps := testDeps()
	deps.Config = Config{
		Env:                     EnvDev,
		AllowedOrigins:          []string{"*"},
		EnableDevModeration:     true,
		DevModerationToken:      "socket-test-token",
		JSONBodyLimitBytes:      1024,
		HTTPJSONBodyReadTimeout: 50 * time.Millisecond,
	}
	// A non-nil store activates the response hydrator. The slow-body failure
	// remains non-successful, so hydration never needs to query this store.
	deps.ProfileCustomisationStore = api.NewProfileCustomisationStore(nil)

	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	address, stop := startRouteSocketServer(t, mux)
	defer stop()

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection,
		"POST /v1/dev/moderation/ozone-events HTTP/1.1\r\n"+
			"Host: appview.example\r\n"+
			"Content-Type: application/json\r\n"+
			"X-Craftsky-Dev-Moderation-Token: socket-test-token\r\n"+
			"Idempotency-Key: route-body-deadline-0001\r\n"+
			"Transfer-Encoding: chunked\r\n\r\n"+
			"10\r\n{\"subject\":"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodPost})
	if err != nil {
		t.Fatalf("read slow-body response: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestTimeout {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("slow-body status = %d, want 408; body=%s", response.StatusCode, body)
	}
	if !response.Close {
		t.Fatalf("slow-body response did not close the connection: headers=%v", response.Header)
	}
}

func TestAddRoutesUnsupportedMediaClosesUnreadBodySocket(t *testing.T) {
	deps := testDeps()
	deps.Config = Config{
		Env:                 EnvDev,
		AllowedOrigins:      []string{"*"},
		EnableDevModeration: true,
		DevModerationToken:  "socket-test-token",
		JSONBodyLimitBytes:  1024,
	}
	deps.ProfileCustomisationStore = api.NewProfileCustomisationStore(nil)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	address, stop := startRouteSocketServer(t, mux)
	defer stop()

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection,
		"POST /v1/dev/moderation/ozone-events HTTP/1.1\r\n"+
			"Host: appview.example\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: 100\r\n\r\n"+
			"unread"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodPost})
	if err != nil {
		t.Fatalf("read unsupported-media response: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnsupportedMediaType {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("unsupported-media status = %d, want 415; body=%s", response.StatusCode, body)
	}
	if !response.Close {
		t.Fatalf("unsupported-media response did not close the connection: headers=%v", response.Header)
	}
}

func startRouteSocketServer(t *testing.T, handler http.Handler) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       time.Second,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.Serve(listener)
	}()
	return listener.Addr().String(), func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownContext)
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("route socket server did not stop")
		}
	}
}
