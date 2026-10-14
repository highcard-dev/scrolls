// Publishes the existing explicit catalog through the shared Core lifecycle.
// Registry credentials can write private staging only; Core owns finalization.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type publisher struct {
	base, token, owner, host, repositorySuffix string
	public                                     bool
	client                                     *http.Client
	authURL                                    string
}

var exact = regexp.MustCompile(`^[^@\s]+@sha256:[a-f0-9]{64}$`)

func configured() (*publisher, error) {
	base := strings.TrimRight(os.Getenv("SCROLL_LIFECYCLE_URL"), "/")
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "druid-gs"))) {
		return nil, errors.New("SCROLL_LIFECYCLE_URL must be HTTPS (HTTP allowed only for local development)")
	}
	p := &publisher{base: base, token: os.Getenv("SCROLL_LIFECYCLE_TOKEN"), owner: os.Getenv("SCROLL_LIFECYCLE_OWNER"), host: os.Getenv("SCROLL_REGISTRY_HOST"), repositorySuffix: os.Getenv("SCROLL_LIFECYCLE_REPOSITORY_SUFFIX"), public: os.Getenv("SCROLL_LIFECYCLE_PUBLISH") == "1",
		client: &http.Client{Timeout: 70 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	if p.owner == "" || p.host == "" || os.Getenv("SCROLL_REGISTRY_NAMESPACE") != p.owner {
		return nil, errors.New("expected owner, registry host and matching private staging namespace are required")
	}
	if p.public && os.Getenv("SCROLL_LIFECYCLE_REVIEWED") != "1" {
		return nil, errors.New("public publication requires SCROLL_LIFECYCLE_REVIEWED=1 after content review")
	}
	// Never reuse production admin/public-project credentials for staging.
	if !strings.HasPrefix(os.Getenv("SCROLL_REGISTRY_USER"), "robot$"+p.owner+"+") {
		return nil, errors.New("registry credentials must belong to the expected owner's private project")
	}
	if p.token == "" {
		auth, err := url.Parse(strings.TrimRight(os.Getenv("SCROLL_AUTH_URL"), "/"))
		if err != nil || auth.Host != u.Host || auth.Scheme != u.Scheme || auth.User != nil || auth.RawQuery != "" || auth.Fragment != "" {
			return nil, errors.New("SCROLL_AUTH_URL must use the same trusted origin as Core")
		}
		email, password := os.Getenv("SCROLL_TEAM_EMAIL"), os.Getenv("SCROLL_TEAM_PASSWORD")
		if email == "" || password == "" {
			return nil, errors.New("provide a fresh lifecycle token or dedicated Team sign-in credentials")
		}
		p.authURL = auth.String()
		p.client.Jar, _ = cookiejar.New(nil)
		if err := p.authRequest(http.MethodPost, "/sign-in/email", map[string]string{"email": email, "password": password}, nil); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Use normal account sign-in and refreshed JWTs, not a Team-specific auth bypass.
// Session cookies stay in memory and are revoked on exit.
func (p *publisher) authRequest(method, path string, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, p.authURL+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("invalid auth request")
	}
	req.Header.Set("Content-Type", "application/json")
	if origin, err := url.Parse(p.authURL); err == nil {
		req.Header.Set("Origin", origin.Scheme+"://"+origin.Host)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return errors.New("publisher authentication failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var failure struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1024)).Decode(&failure)
		if !regexp.MustCompile(`^[A-Z_]{1,64}$`).MatchString(failure.Code) {
			failure.Code = "AUTH_FAILED"
		}
		return fmt.Errorf("publisher authentication rejected: HTTP %d (%s)", resp.StatusCode, failure.Code)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(out)
	}
	return nil
}

func (p *publisher) request(method, path string, body, out any) error {
	if p.authURL != "" {
		var jwt struct {
			Token string `json:"token"`
		}
		if err := p.authRequest(http.MethodGet, "/token", nil, &jwt); err != nil {
			return err
		}
		if jwt.Token == "" {
			return errors.New("authentication returned no owner JWT")
		}
		p.token = jwt.Token
	}
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, p.base+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("invalid lifecycle request")
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return errors.New("lifecycle request failed; inspect installed/published state before retrying")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("lifecycle request rejected: HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(out)
	}
	return nil
}

func (p *publisher) preflight() error {
	var project struct {
		Registry string `json:"registry"`
		Project  string `json:"project"`
	}
	// The shared operation ensures this is the authenticated actor's PRIVATE
	// project. A supplied owner name alone is never treated as authorization.
	if err := p.request(http.MethodGet, "/v1/registry/project", nil, &project); err != nil {
		return err
	}
	if project.Registry != p.host || project.Project != p.host+"/"+p.owner {
		return errors.New("authenticated identity or registry differs from the expected publisher")
	}
	return nil
}

func (p *publisher) publish(command []string, run func([]string) error) error {
	if len(command) < 4 || command[1] != "push" || command[2] == "category" {
		return errors.New("expected a rendered artifact push command")
	}
	ref := command[2]
	prefix := p.host + "/" + p.owner + "/"
	if !strings.HasPrefix(ref, prefix) {
		return errors.New("artifact must target the verified private staging project")
	}
	nameTag := strings.TrimPrefix(ref, prefix)
	parts := strings.Split(nameTag, ":")
	if len(parts) != 2 || strings.Contains(parts[0], "/") || !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`).MatchString(parts[0]) || !regexp.MustCompile(`^[\w][\w.-]{0,127}$`).MatchString(parts[1]) {
		return errors.New("invalid repository or explicit revision tag")
	}
	name := parts[0] + p.repositorySuffix
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`).MatchString(name) {
		return errors.New("invalid final repository name")
	}
	if err := p.preflight(); err != nil {
		return err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	// Each invocation owns its staging tag, so another job cannot replace the
	// mutable selection between our push and the single digest resolution.
	stagedRef := prefix + parts[0] + ":build-" + hex.EncodeToString(nonce)
	stagedCommand := append([]string(nil), command...)
	stagedCommand[2] = stagedRef
	// Final reusable revisions embed presentation instead of depending on a
	// separately mutable category artifact. Staging supplies inherited .meta.
	stagedCommand = append(stagedCommand, "--pack-meta")
	if err := run(stagedCommand); err != nil {
		return err
	}
	var staged struct {
		Artifact string `json:"artifact"`
	}
	if err := p.request(http.MethodGet, "/v1/registry/scrolls/revision?artifact="+url.QueryEscape(stagedRef), nil, &staged); err != nil {
		return err
	}
	if !exact.MatchString(staged.Artifact) || !strings.HasPrefix(staged.Artifact, prefix+parts[0]+"@") {
		return errors.New("resolved staging revision differs from selected repository")
	}
	var imported struct {
		Artifact string `json:"artifact"`
	}
	input := map[string]any{"repository": name, "tag": parts[1], "releaseArtifact": staged.Artifact, "reviewedPublicContent": p.public}
	if err := p.request(http.MethodPost, "/v1/registry/scrolls/releases/import", input, &imported); err != nil {
		return err
	}
	ownerHash := sha256.Sum256([]byte(p.owner))
	canonical := fmt.Sprintf("%s/scroll-%x-%s/%s", p.host, ownerHash, name, name)
	if !exact.MatchString(imported.Artifact) || !strings.HasPrefix(imported.Artifact, canonical+"@") {
		return errors.New("Core did not return an exact finalized revision")
	}
	if p.public {
		digest := strings.SplitN(imported.Artifact, "@", 2)[1]
		if err := p.request(http.MethodPost, "/v1/registry/scrolls/publish", map[string]any{"repository": name, "revision": digest, "reviewedPublicContent": true}, nil); err != nil {
			return err
		}
	}
	fmt.Println(imported.Artifact)
	return nil
}

func runPublisher() error {
	p, err := configured()
	if err == nil && p.authURL != "" {
		defer p.authRequest(http.MethodPost, "/sign-out", map[string]any{}, nil)
	}
	if err == nil {
		if len(os.Args) == 2 && os.Args[1] == "preflight" {
			err = p.preflight()
		} else {
			args := os.Args[1:]
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}
			err = p.publish(args, func(command []string) error {
				cmd := exec.Command(command[0], command[1:]...)
				for _, entry := range os.Environ() {
					if strings.HasPrefix(entry, "SCROLL_TEAM_PASSWORD=") || strings.HasPrefix(entry, "SCROLL_TEAM_EMAIL=") || strings.HasPrefix(entry, "SCROLL_LIFECYCLE_TOKEN=") {
						continue
					}
					cmd.Env = append(cmd.Env, entry)
				}
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				return cmd.Run()
			})
		}
	}
	return err
}

func main() {
	if err := runPublisher(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
