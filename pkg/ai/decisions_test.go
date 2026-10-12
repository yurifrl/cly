package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// jevServer builds a Decisions stub returning the given JSON with status.
func jevServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const jevOKResponse = `{
  "id": "r1",
  "model": "jev-model",
  "answers": {
    "s0": {"type": "choice", "choice": "own", "probabilities": {"own": 0.9, "s1": 0.1}},
    "s1": {"type": "noul", "noul": 0.75}
  },
  "usage": {"input_tokens": 10, "output_tokens": 5, "cost": 0.01}
}`

func TestDecideWire(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody decisionsRequestWire
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jevOKResponse))
	}))
	t.Cleanup(srv.Close)

	c, err := newDecisionsClient([]decisionsEndpoint{
		{name: "t", model: "jev-model", baseURL: srv.URL, apiKey: "k1"},
	}, srv.Client())
	if err != nil {
		t.Fatalf("newDecisionsClient: %v", err)
	}
	res, err := c.Decide(context.Background(),
		map[string]string{"repo": "cly"},
		[]DecisionQuestion{
			{ID: "s0", Type: "choice", Instructions: "pick", Criteria: map[string]string{"own": "its own commit"}},
			{ID: "s1", Type: "noul"},
		})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if gotPath != decisionsAPIPath {
		t.Errorf("path = %q, want %q", gotPath, decisionsAPIPath)
	}
	if gotAuth != "Bearer k1" {
		t.Errorf("auth = %q, want Bearer k1", gotAuth)
	}
	if gotBody.Model != "jev-model" {
		t.Errorf("model = %q, want jev-model (model must come from config)", gotBody.Model)
	}
	if gotBody.State["repo"] != "cly" {
		t.Errorf("state not sent: %+v", gotBody.State)
	}
	a := res.Answers["s0"]
	if a.Type != "choice" || a.Choice != "own" || a.Probabilities["s1"] != 0.1 {
		t.Errorf("s0 answer = %+v", a)
	}
	if n := res.Answers["s1"].Noul; n != 0.75 {
		t.Errorf("s1 noul = %v, want 0.75", n)
	}
	if res.Cost != 0.01 {
		t.Errorf("cost = %v, want 0.01", res.Cost)
	}
}

func TestDecideFailsOverWithinJevPool(t *testing.T) {
	var second atomic.Int32
	bad := jevServer(t, http.StatusInternalServerError, `{"error":"down"}`)
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		second.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jevOKResponse))
	}))
	t.Cleanup(good.Close)

	c, err := newDecisionsClient([]decisionsEndpoint{
		{name: "bad", model: "m-bad", baseURL: bad.URL, apiKey: "k"},
		{name: "good", model: "m-good", baseURL: good.URL, apiKey: "k"},
	}, http.DefaultClient)
	if err != nil {
		t.Fatalf("newDecisionsClient: %v", err)
	}
	res, err := c.Decide(context.Background(), nil,
		[]DecisionQuestion{{ID: "s0", Type: "noul"}})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if second.Load() != 1 {
		t.Fatalf("second endpoint not tried (hits=%d); failover broken", second.Load())
	}
	if len(res.Answers) != 2 {
		t.Errorf("answers = %d, want 2", len(res.Answers))
	}
}

func TestDecideAllEndpointsDown(t *testing.T) {
	bad1 := jevServer(t, http.StatusInternalServerError, "boom")
	bad2 := jevServer(t, http.StatusUnauthorized, "denied")
	c, err := newDecisionsClient([]decisionsEndpoint{
		{name: "b1", model: "m", baseURL: bad1.URL, apiKey: "k"},
		{name: "b2", model: "m", baseURL: bad2.URL, apiKey: "k"},
	}, http.DefaultClient)
	if err != nil {
		t.Fatalf("newDecisionsClient: %v", err)
	}
	if _, err := c.Decide(context.Background(), nil,
		[]DecisionQuestion{{ID: "s0", Type: "noul"}}); err == nil {
		t.Fatal("expected error when every jev endpoint fails")
	}
}

func TestDecideRejectsDuplicateIDs(t *testing.T) {
	c, err := newDecisionsClient([]decisionsEndpoint{
		{name: "t", model: "m", baseURL: "http://unused", apiKey: "k"},
	}, nil)
	if err != nil {
		t.Fatalf("newDecisionsClient: %v", err)
	}
	_, err = c.Decide(context.Background(), nil,
		[]DecisionQuestion{{ID: "x", Type: "noul"}, {ID: "x", Type: "noul"}})
	if err == nil {
		t.Fatal("expected duplicate-id error")
	}
}

func TestNewDecisionsClientRequiresPool(t *testing.T) {
	if _, err := newDecisionsClient(nil, nil); err == nil {
		t.Fatal("expected error with no jev endpoints")
	}
}
