package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRef(owner, host, name, tag string) string {
	return fmt.Sprintf("%s/%s/%s:%s", host, owner, name, tag)
}

func TestParseRefRequiresOwnerPrivateProject(t *testing.T) {
	p := &publisher{owner: "owner-1", host: "registry.example:5000"}
	ref := testRef(p.owner, p.host, "example", "v1")
	name, got, err := p.parseRef(ref)
	if err != nil || name != "example" || got != ref {
		t.Fatalf("owned direct push rejected: %q %q %v", name, got, err)
	}
	for _, invalid := range []string{
		testRef("another-owner", p.host, "example", "v1"),
		"other.example/project/example:v1",
		strings.Replace(ref, "/example:v1", "/nested/example:v1", 1),
		strings.TrimSuffix(ref, ":v1") + ":invalid:tag",
	} {
		if _, _, err := p.parseRef(invalid); err == nil {
			t.Errorf("accepted unsafe push manifest reference: %s", invalid)
		}
	}
}

func TestPublishManifestUsesReviewedSnapshotAndOneOperationPerRepository(t *testing.T) {
	t.Setenv("SCROLL_LIFECYCLE_REVIEWED", "1")
	owner, host, name := "owner-1", "registry.example:5000", "example"
	first := testRef(owner, host, name, "v1")
	second := testRef(owner, host, name, "v2")
	repository := first[:strings.LastIndex(first, ":")]
	digest := "sha256:" + strings.Repeat("a", 64)
	snapshot := []map[string]any{{"digest": digest, "tags": []string{"v1", "v2"}}}
	manifest := filepath.Join(t.TempDir(), "pushed.txt")
	if err := os.WriteFile(manifest, []byte(first+"\n"+second+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	resolved, previewed, published := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sign-in/email":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "fixture", Path: "/"})
		case "/token":
			if _, err := r.Cookie("session"); err != nil {
				t.Error("missing owner session")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "owner-jwt"})
		case "/v1/registry/project":
			if r.Header.Get("Authorization") != "Bearer owner-jwt" {
				t.Error("missing owner token")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"registry": host, "project": host + "/" + owner})
		case "/v1/registry/scrolls/revision":
			resolved++
			if r.URL.Query().Get("artifact") != first && r.URL.Query().Get("artifact") != second {
				t.Error("resolved wrong pushed ref")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"repository": repository, "digest": digest, "artifact": repository + "@" + digest})
		case "/v1/registry/repositories/example/publication-preview":
			previewed++
			_ = json.NewEncoder(w).Encode(snapshot)
		case "/v1/registry/scrolls/publish":
			published++
			var input struct {
				Repository string           `json:"repository"`
				Snapshot   []map[string]any `json:"expectedSnapshot"`
				Reviewed   bool             `json:"reviewedPublicContent"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			got, _ := json.Marshal(input.Snapshot)
			want, _ := json.Marshal(snapshot)
			if input.Repository != name || string(got) != string(want) || !input.Reviewed {
				t.Errorf("unexpected publication input: %+v", input)
			}
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	p := &publisher{base: server.URL, auth: server.URL, owner: owner, host: host, manifest: manifest, client: &http.Client{Jar: jar}}
	if err := p.authRequest(http.MethodPost, "/sign-in/email", map[string]string{"email": "owner@example.com", "password": "fixture"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.preflight(); err != nil {
		t.Fatal(err)
	}
	if err := p.publishManifest(); err != nil {
		t.Fatal(err)
	}
	if resolved != 2 || previewed != 1 || published != 1 {
		t.Fatalf("expected two resolves, one preview and one publish, got %d, %d and %d", resolved, previewed, published)
	}
}

func TestPublishManifestRequiresReview(t *testing.T) {
	t.Setenv("SCROLL_LIFECYCLE_REVIEWED", "")
	if err := (&publisher{}).publishManifest(); err == nil {
		t.Fatal("publication without content review accepted")
	}
}

func TestPublishManifestRejectsUnsafePreview(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid digest", "missing tag", "moved tag", "conflict"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("SCROLL_LIFECYCLE_REVIEWED", "1")
			digest := "sha256:" + strings.Repeat("a", 64)
			ref := testRef("owner", "registry.example", "example", "v1")
			manifest := filepath.Join(t.TempDir(), "pushed.txt")
			if err := os.WriteFile(manifest, []byte(ref+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			published := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/token":
					_ = json.NewEncoder(w).Encode(map[string]string{"token": "fixture"})
				case "/v1/registry/repositories/example/publication-preview":
					snapshot := []publicationArtifact{{Digest: digest, Tags: []string{"v1"}}}
					switch scenario {
					case "empty":
						snapshot = nil
					case "invalid digest":
						snapshot[0].Digest = "latest"
					case "missing tag":
						snapshot[0].Tags = []string{"v2"}
					case "moved tag":
						snapshot[0].Digest = "sha256:" + strings.Repeat("b", 64)
					}
					_ = json.NewEncoder(w).Encode(snapshot)
				case "/v1/registry/scrolls/revision":
					_ = json.NewEncoder(w).Encode(map[string]string{"repository": "registry.example/owner/example", "digest": digest, "artifact": "registry.example/owner/example@" + digest})
				case "/v1/registry/scrolls/publish":
					published++
					http.Error(w, "changed since review", http.StatusConflict)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			p := &publisher{base: server.URL, auth: server.URL, owner: "owner", host: "registry.example", manifest: manifest, client: server.Client()}
			if err := p.publishManifest(); err == nil {
				t.Fatal("unsafe publication accepted")
			}
			want := 0
			if scenario == "conflict" {
				want = 1
			}
			if published != want {
				t.Fatalf("publication requests: got %d, want %d; conflicts must not auto-retry", published, want)
			}
		})
	}
}
