package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSharedLifecyclePrivateStagingAndExplicitPublication(t *testing.T) {
	for _, public := range []bool{false, true} {
		t.Run(fmt.Sprint(public), func(t *testing.T) {
			var staged string
			var imported, published int
			name := "scroll-example"
			suffix := ""
			if !public {
				suffix = "-pr12"
			}
			name += suffix
			canonical := fmt.Sprintf("registry.test/scroll-%x-%s/%s", sha256.Sum256([]byte("team-owner")), name, name)
			digest := "sha256:" + strings.Repeat("a", 64)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer fixture" {
					t.Error("owner authorization missing")
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v1/registry/project":
					fmt.Fprint(w, `{"registry":"registry.test","project":"registry.test/team-owner"}`)
				case "/v1/registry/scrolls/revision":
					if staged == "" || r.URL.Query().Get("artifact") != staged {
						t.Error("did not freeze own staging tag")
					}
					json.NewEncoder(w).Encode(map[string]string{"artifact": "registry.test/team-owner/scroll-example@" + digest})
				case "/v1/registry/scrolls/releases/import":
					imported++
					var input struct {
						Repository, Tag, ReleaseArtifact string
						ReviewedPublicContent            bool
					}
					json.NewDecoder(r.Body).Decode(&input)
					if input.Repository != name || input.Tag != "v1-commit" || input.ReleaseArtifact != "registry.test/team-owner/scroll-example@"+digest || input.ReviewedPublicContent != public {
						t.Errorf("wrong import contract: %+v", input)
					}
					json.NewEncoder(w).Encode(map[string]string{"artifact": canonical + "@" + digest})
				case "/v1/registry/scrolls/publish":
					published++
					var input struct {
						Repository, Revision  string
						ReviewedPublicContent bool
					}
					json.NewDecoder(r.Body).Decode(&input)
					if input.Repository != name || input.Revision != digest || !input.ReviewedPublicContent {
						t.Error("publication lost exact revision/review")
					}
					fmt.Fprint(w, `{}`)
				default:
					t.Error("unexpected request")
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			p := &publisher{base: server.URL, client: server.Client(), token: "fixture", owner: "team-owner", host: "registry.test", public: public, repositorySuffix: suffix}
			command := []string{"druid", "push", "registry.test/team-owner/scroll-example:v1-commit", "./scroll"}
			err := p.publish(command, func(args []string) error {
				staged = args[2]
				if !strings.HasPrefix(staged, "registry.test/team-owner/scroll-example:build-") {
					t.Fatal("push not isolated in private staging")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if imported != 1 || (public && published != 1) || (!public && published != 0) {
				t.Fatal("wrong lifecycle calls")
			}
			if command[2] != "registry.test/team-owner/scroll-example:v1-commit" {
				t.Fatal("caller catalog mutated")
			}
		})
	}
}

func TestWrongIdentityStopsBeforeRegistryPush(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"registry":"registry.test","project":"registry.test/other"}`)
	}))
	defer server.Close()
	p := &publisher{base: server.URL, client: server.Client(), owner: "team-owner", host: "registry.test"}
	called := false
	err := p.publish([]string{"druid", "push", "registry.test/team-owner/example:v1", "./scroll"}, func([]string) error { called = true; return nil })
	if err == nil || called {
		t.Fatal("wrong identity was allowed to push")
	}
}

func TestFailedPushNeverImports(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"registry":"registry.test","project":"registry.test/team-owner"}`)
	}))
	defer server.Close()
	p := &publisher{base: server.URL, client: server.Client(), owner: "team-owner", host: "registry.test"}
	err := p.publish([]string{"druid", "push", "registry.test/team-owner/example:v1", "./scroll"}, func([]string) error { return errors.New("push failed") })
	if err == nil || calls != 1 {
		t.Fatal("failed push reached lifecycle import")
	}
}

func TestConfigurationRejectsAdminCredentialsAndUnreviewedPublication(t *testing.T) {
	t.Setenv("SCROLL_LIFECYCLE_URL", "https://api.druid.gg/core")
	t.Setenv("SCROLL_LIFECYCLE_TOKEN", "fixture")
	t.Setenv("SCROLL_LIFECYCLE_OWNER", "team-owner")
	t.Setenv("SCROLL_REGISTRY_HOST", "registry.test")
	t.Setenv("SCROLL_REGISTRY_NAMESPACE", "team-owner")
	t.Setenv("SCROLL_REGISTRY_USER", "admin")
	if _, err := configured(); err == nil {
		t.Fatal("admin credentials accepted")
	}
	t.Setenv("SCROLL_REGISTRY_USER", "robot$team-owner+ci")
	t.Setenv("SCROLL_LIFECYCLE_PUBLISH", "1")
	t.Setenv("SCROLL_LIFECYCLE_REVIEWED", "0")
	if _, err := configured(); err == nil {
		t.Fatal("unreviewed public import accepted")
	}
	t.Setenv("SCROLL_LIFECYCLE_REVIEWED", "1")
	if _, err := configured(); err != nil {
		t.Fatal(err)
	}
}

func TestDedicatedAccountRefreshesJWTUsingInMemorySession(t *testing.T) {
	tokens := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/v2/sign-in/email":
			var input map[string]string
			json.NewDecoder(r.Body).Decode(&input)
			if input["email"] != "team@example.test" || input["password"] != "fixture-password" {
				t.Error("wrong dedicated sign-in credentials")
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "fixture-session", Path: "/"})
			fmt.Fprint(w, `{}`)
		case "/auth/v2/token":
			if cookie, err := r.Cookie("session"); err != nil || cookie.Value != "fixture-session" {
				t.Error("session cookie missing")
			}
			tokens++
			fmt.Fprintf(w, `{"token":"refreshed-%d"}`, tokens)
		case "/core/v1/registry/project":
			if r.Header.Get("Authorization") != fmt.Sprintf("Bearer refreshed-%d", tokens) {
				t.Error("fresh owner JWT missing")
			}
			fmt.Fprint(w, `{"registry":"registry.test","project":"registry.test/team-owner"}`)
		default:
			t.Error("unexpected request")
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	t.Setenv("SCROLL_LIFECYCLE_URL", server.URL+"/core")
	t.Setenv("SCROLL_AUTH_URL", server.URL+"/auth/v2")
	t.Setenv("SCROLL_LIFECYCLE_TOKEN", "")
	t.Setenv("SCROLL_TEAM_EMAIL", "team@example.test")
	t.Setenv("SCROLL_TEAM_PASSWORD", "fixture-password")
	t.Setenv("SCROLL_LIFECYCLE_OWNER", "team-owner")
	t.Setenv("SCROLL_REGISTRY_NAMESPACE", "team-owner")
	t.Setenv("SCROLL_REGISTRY_HOST", "registry.test")
	t.Setenv("SCROLL_REGISTRY_USER", "robot$team-owner+ci")
	t.Setenv("SCROLL_LIFECYCLE_PUBLISH", "0")
	p, err := configured()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.preflight(); err != nil {
		t.Fatal(err)
	}
	if err := p.preflight(); err != nil {
		t.Fatal(err)
	}
	if tokens != 2 {
		t.Fatal("owner JWT was not refreshed before each lifecycle request")
	}
}

func TestPublisherNeverFollowsCredentialedRedirects(t *testing.T) {
	forwarded := false
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer other.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	t.Setenv("SCROLL_LIFECYCLE_URL", server.URL)
	t.Setenv("SCROLL_LIFECYCLE_TOKEN", "fixture")
	t.Setenv("SCROLL_LIFECYCLE_OWNER", "team-owner")
	t.Setenv("SCROLL_REGISTRY_NAMESPACE", "team-owner")
	t.Setenv("SCROLL_REGISTRY_HOST", "registry.test")
	t.Setenv("SCROLL_REGISTRY_USER", "robot$team-owner+ci")
	t.Setenv("SCROLL_LIFECYCLE_PUBLISH", "0")
	p, err := configured()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.preflight(); err == nil {
		t.Fatal("redirect accepted")
	}
	if forwarded {
		t.Fatal("credentials reached redirect target")
	}
}
