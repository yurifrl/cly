package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecideWire(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"gen-1","model":"typesafe/jev-1.13","provider":"aihub","answers":{
			"q1":{"type":"choice","choice":"b2","confidence":0.7,"probabilities":{"b1":0.2,"b2":0.8}},
			"q2":{"type":"noul","noul":0.93},
			"q3":{"type":"score","score":{"position":2.5,"confidence":0.6}}
		},"usage":{"input_tokens":100,"output_tokens":10,"cost":0.001}}`))
	}))
	defer srv.Close()

	t.Setenv("AIHUB_API_KEY", "test-key")
	t.Setenv("DECISIONS_BASE_URL", srv.URL)
	c, err := NewDecisionsClient()
	if err != nil {
		t.Fatalf("NewDecisionsClient: %v", err)
	}

	res, err := c.Decide(context.Background(), "aihub/jev-latest",
		map[string]string{"repo": "cly"},
		[]DecisionQuestion{
			{ID: "q1", Type: "choice", Instructions: "pick", Criteria: map[string]string{"b1": "one", "b2": "two"}},
			{ID: "q2", Type: "noul", Instructions: "is it so?"},
			{ID: "q3", Type: "score", Instructions: "rank", Criteria: []string{"low", "mid", "high"}},
		})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}

	if gotPath != "/api/alpha/decisions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotBody["model"] != "aihub/jev-latest" {
		t.Errorf("model = %v", gotBody["model"])
	}
	if len(gotBody["questions"].(map[string]any)) != 3 {
		t.Errorf("questions = %v", gotBody["questions"])
	}

	if res.Answers["q1"].Choice != "b2" || res.Answers["q1"].Probabilities["b2"] != 0.8 {
		t.Errorf("q1 = %+v", res.Answers["q1"])
	}
	if res.Answers["q2"].Noul != 0.93 {
		t.Errorf("q2 = %+v", res.Answers["q2"])
	}
	if res.Answers["q3"].ScorePosition != 2.5 || res.Answers["q3"].Confidence != 0.6 {
		t.Errorf("q3 = %+v", res.Answers["q3"])
	}
	if res.Cost != 0.001 {
		t.Errorf("cost = %v", res.Cost)
	}
}

func TestNewDecisionsClientRequiresKey(t *testing.T) {
	t.Setenv("AIHUB_API_KEY", "")
	if _, err := NewDecisionsClient(); err == nil {
		t.Fatal("expected error with unset key")
	}
}

func TestDecideRejectsDuplicateIDs(t *testing.T) {
	t.Setenv("AIHUB_API_KEY", "k")
	c, err := NewDecisionsClient()
	if err != nil {
		t.Fatalf("NewDecisionsClient: %v", err)
	}
	_, err = c.Decide(context.Background(), "m", nil,
		[]DecisionQuestion{{ID: "x", Type: "noul"}, {ID: "x", Type: "noul"}})
	if err == nil {
		t.Fatal("expected duplicate-id error")
	}
}
