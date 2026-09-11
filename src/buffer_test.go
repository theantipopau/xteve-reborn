package src

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveReachableStreamURLPrefersWorkingPrimary(t *testing.T) {

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backup.Close()

	streamInfo := StreamInfo{URL: primary.URL, BackupURL1: backup.URL}

	got := resolveReachableStreamURL(streamInfo)
	if got != primary.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the primary URL %q (it was reachable, no need to fail over)", got, primary.URL)
	}
}

func TestResolveReachableStreamURLFallsBackToFirstWorkingBackup(t *testing.T) {

	deadPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	deadBackup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer deadBackup.Close()

	workingBackup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer workingBackup.Close()

	streamInfo := StreamInfo{
		URL:        deadPrimary.URL,
		BackupURL1: deadBackup.URL,
		BackupURL2: workingBackup.URL,
		BackupURL3: "http://127.0.0.1:1/unreachable",
	}

	got := resolveReachableStreamURL(streamInfo)
	if got != workingBackup.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the first working backup %q", got, workingBackup.URL)
	}
}

func TestResolveReachableStreamURLFallsBackToPrimaryWhenNothingWorks(t *testing.T) {

	deadPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	streamInfo := StreamInfo{
		URL:        deadPrimary.URL,
		BackupURL1: "http://127.0.0.1:1/unreachable",
	}

	// Nothing is reachable, so a channel with no working backup must behave
	// exactly as it did before backup channels existed: use the primary and
	// let the existing downstream error handling take over.
	got := resolveReachableStreamURL(streamInfo)
	if got != deadPrimary.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the primary URL %q unchanged when nothing is reachable", got, deadPrimary.URL)
	}
}

func TestIsStreamURLReachable(t *testing.T) {

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()

	if !isStreamURLReachable(ok.URL) {
		t.Error("isStreamURLReachable() = false for a 200 OK server, want true")
	}
	if isStreamURLReachable(notFound.URL) {
		t.Error("isStreamURLReachable() = true for a 404 server, want false")
	}
	if isStreamURLReachable("") {
		t.Error("isStreamURLReachable() = true for an empty URL, want false")
	}
	if isStreamURLReachable("http://127.0.0.1:1/unreachable") {
		t.Error("isStreamURLReachable() = true for an unreachable address, want false")
	}
}
