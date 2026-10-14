package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Opt-in localhost only. Creates an isolated publisher account/project and one
// tiny Scroll. It neither deploys a workload nor consumes shared game ports.
func TestLocalAuthenticatedPublisher(t *testing.T) {
	if os.Getenv("SCROLL_LOCAL_LIFECYCLE_TEST") != "1" {
		t.Skip("local live stack opt-in")
	}
	bin := os.Getenv("DRUID_BIN")
	if bin == "" {
		t.Fatal("DRUID_BIN is required")
	}
	jar, _ := cookiejar.New(nil)
	p := &publisher{base: "http://localhost:3000/api/core", authURL: "http://localhost:3000/api/auth/v2", host: "druid-gs:8088", client: &http.Client{Jar: jar, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	suffix := hex.EncodeToString(nonce)
	var account struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := p.authRequest(http.MethodPost, "/sign-up/email", map[string]any{"email": "lifecycle-" + suffix + "@example.invalid", "password": suffix + "-Fixture!", "name": "Local lifecycle acceptance", "username": "lifecycle" + suffix[:12]}, &account); err != nil {
		t.Fatal(err)
	}
	p.owner = account.User.ID
	if p.owner == "" {
		t.Fatal("signup returned no fixture identity")
	}
	t.Logf("local fixture identity: %s", p.owner)
	t.Cleanup(func() { cleanupLocalPublisher(t, p.owner) })
	t.Cleanup(func() { _ = p.authRequest(http.MethodPost, "/sign-out", map[string]any{}, nil) })
	// Exercise the same normal dedicated-account sign-in used by CI, not just
	// the initial signup session returned by fixture setup.
	if err := p.authRequest(http.MethodPost, "/sign-out", map[string]any{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.authRequest(http.MethodPost, "/sign-in/email", map[string]string{"email": "lifecycle-" + suffix + "@example.invalid", "password": suffix + "-Fixture!"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.preflight(); err != nil {
		t.Fatal(err)
	}
	var robot struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := p.request(http.MethodPost, "/v1/registry/credentials", map[string]string{"name": "lifecycle-fixture"}, &robot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.request(http.MethodDelete, fmt.Sprintf("/v1/registry/credentials/%d", robot.ID), nil, nil); err != nil {
			t.Error("fixture credential revocation failed")
		}
	})
	cliHome := t.TempDir()
	run := func(args []string) error {
		cmd := exec.Command(args[0], args[1:]...)
		for _, value := range os.Environ() {
			if !strings.HasPrefix(value, "HOME=") {
				cmd.Env = append(cmd.Env, value)
			}
		}
		cmd.Env = append(cmd.Env, "HOME="+cliHome, "DRUID_REGISTRY_PLAIN_HTTP=true", "SOURCE_DATE_EPOCH=0")
		output, err := cmd.CombinedOutput()
		if err != nil && len(args) > 1 && args[1] != "login" {
			// Fixture-only diagnostics: never print auth/login output or full logs.
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(line, "Error:") {
					t.Log(strings.ReplaceAll(strings.ReplaceAll(line, robot.Password, "<REDACTED>"), p.token, "<REDACTED>"))
				}
			}
		}
		return err
	}
	if err := run([]string{bin, "login", "--host", p.host, "--user", robot.Username, "--password", robot.Password}); err != nil {
		t.Fatal("fixture registry login failed")
	}
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, ".meta"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, ".meta", "en-US.md"), []byte("---\nname: Local authored fixture\n---\nNo customer content.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "scroll.yaml"), []byte("name: fixture\ndesc: Local authored fixture\nversion: 1.0.0\napp_version: '1.0.0'\ncommands:\n  start:\n    run: persistent\n    procedures:\n      - id: fixture\n        command: [sleep, '600']\n        image: busybox:1.36\n"), 0644); err != nil {
		t.Fatal(err)
	}
	command := []string{bin, "push", p.host + "/" + p.owner + "/example:accepted-fixture", source, "--category", "fixture"}
	if err := p.publish(command, run); err != nil {
		t.Fatal(err)
	}
	// Repeat exactly through the real authenticated public operation. It must
	// verify the existing finalized bytes, not overwrite the revision tag.
	p.public = true
	if err := p.publish(command, run); err != nil {
		t.Fatal(err)
	}
	t.Log("authenticated private staging, exact retry and shared publication passed")
	// Assert publication through Harbor's public metadata without owner cookies.
	projectName := fmt.Sprintf("scroll-%x-example", sha256.Sum256([]byte(p.owner)))
	response, err := http.Get("http://druid-gs:8088/api/v2.0/projects?page_size=100&name=" + projectName)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal("anonymous public project listing failed")
	}
	var projects []struct {
		Name     string `json:"name"`
		Metadata struct {
			Public string `json:"public"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(response.Body).Decode(&projects); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, project := range projects {
		if project.Name == projectName && project.Metadata.Public == "true" {
			found = true
		}
	}
	if !found {
		t.Fatal("public fixture was not listed")
	}
}

func cleanupLocalPublisher(t *testing.T, owner string) {
	t.Helper()
	if !regexp.MustCompile(`^[a-f0-9-]{36}$`).MatchString(owner) {
		t.Error("invalid fixture owner; cleanup refused")
		return
	}
	fixture := fmt.Sprintf(`SELECT id::text FROM auth."user" WHERE id='%s' AND name='Local lifecycle acceptance' AND email ~ '^lifecycle-[a-f0-9]{32}@example[.]invalid$'`, owner)
	verified, err := exec.Command("docker", "exec", "druid-postgres", "psql", "-U", "druid", "-d", "druid", "-At", "-c", fixture).Output()
	if err != nil || strings.TrimSpace(string(verified)) != owner {
		t.Error("exact local fixture identity could not be verified; cleanup refused")
		return
	}
	canonicalProject := fmt.Sprintf("scroll-%x-example", sha256.Sum256([]byte(owner)))
	for _, project := range []string{owner, canonicalProject} {
		for _, path := range []string{"/api/v2.0/projects/" + project + "/repositories/example", "/api/v2.0/projects/" + project} {
			req, _ := http.NewRequest(http.MethodDelete, "http://druid-gs:8088"+path, nil)
			password := os.Getenv("LOCAL_HARBOR_TEST_PASSWORD")
			if password == "" {
				password = "admin"
			}
			req.SetBasicAuth("admin", password)
			req.Header.Set("X-Is-Resource-Name", "true")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error("local Harbor fixture cleanup failed")
				return
			}
			resp.Body.Close()
			if resp.StatusCode != 404 && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
				t.Errorf("local Harbor fixture cleanup HTTP %d", resp.StatusCode)
				return
			}
		}
	}
	// Only this generated fixture identity is eligible. FK failures roll back
	// the entire database cleanup; unrelated/user-owned rows are never targeted.
	sql := fmt.Sprintf(`BEGIN;
DELETE FROM scroll.repository WHERE user_id::text IN (%s);
DELETE FROM core.ledger_entry WHERE user_id::text IN (%s);
DELETE FROM core."user" WHERE identity_id::text IN (%s);
DELETE FROM auth."user" WHERE id::text IN (%s);
COMMIT;`, fixture, fixture, fixture, fixture)
	cmd := exec.Command("docker", "exec", "druid-postgres", "psql", "-U", "druid", "-d", "druid", "-v", "ON_ERROR_STOP=1", "-c", sql)
	if err := cmd.Run(); err != nil {
		t.Error("fixture database cleanup failed; exact identity retained:", owner)
	}
}

func TestCleanupRetainedLocalPublisherFixtures(t *testing.T) {
	ids := os.Getenv("SCROLL_LOCAL_FIXTURE_CLEANUP")
	if ids == "" {
		t.Skip("explicit retained-fixture IDs required")
	}
	for _, id := range strings.Split(ids, ",") {
		cleanupLocalPublisher(t, id)
	}
}
