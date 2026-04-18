package brokerapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubAdapterCreatesPullRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assertGitHubPullRequestRequest(t, r)
		writeGitHubPullRequestResponse(w)
	}))
	defer server.Close()

	adapter := &gitHubPullRequestAdapter{httpClient: server.Client(), apiBaseOverride: server.URL}
	res, err := adapter.CreatePullRequest(context.Background(), gitHubPullRequestTestRequest())
	if err != nil {
		t.Fatalf("CreatePullRequest returned error: %v", err)
	}
	if !called {
		t.Fatal("github adapter did not invoke http endpoint")
	}
	assertGitHubPullRequestResponse(t, res)
}

func gitHubPullRequestTestRequest() gitPullRequestProviderRequest {
	return gitPullRequestProviderRequest{
		Host:       "github.com",
		Repository: "org/repo",
		BaseRef:    "refs/heads/main",
		HeadRef:    "refs/heads/feature",
		HeadOwner:  "org",
		Title:      "Update docs",
		Body:       "Body",
		Token:      "token",
	}
}

func assertGitHubPullRequestRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Fatalf("method=%s, want POST", r.Method)
	}
	if !strings.HasSuffix(r.URL.Path, "/repos/org/repo/pulls") {
		t.Fatalf("path=%s, want /repos/org/repo/pulls", r.URL.Path)
	}
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		t.Fatal("authorization header missing bearer token")
	}
	body, _ := io.ReadAll(r.Body)
	decoded := map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if decoded["head"] != "org:feature" {
		t.Fatalf("head=%v, want org:feature", decoded["head"])
	}
}

func writeGitHubPullRequestResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"number": 42, "html_url": "https://github.example/org/repo/pull/42"}`))
}

func assertGitHubPullRequestResponse(t *testing.T, res gitPullRequestProviderResult) {
	t.Helper()
	if res.Number != 42 {
		t.Fatalf("number=%d, want 42", res.Number)
	}
	if res.URL != "https://github.example/org/repo/pull/42" {
		t.Fatalf("url=%q, want github pull url", res.URL)
	}
}
