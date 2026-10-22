// Publishes direct owner pushes through Core's generic repository operation.
// It never copies, imports, or mutates a selected Scroll artifact.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type publisher struct {
	base, auth, owner, host, email, password, manifest string
	client                                             *http.Client
}

type publicationArtifact struct {
	Digest string   `json:"digest"`
	Tags   []string `json:"tags"`
}

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)
var tagPattern = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)
var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func configured() (*publisher, error) {
	base := strings.TrimRight(os.Getenv("SCROLL_LIFECYCLE_URL"), "/")
	auth := strings.TrimRight(os.Getenv("SCROLL_AUTH_URL"), "/")
	coreURL, err := trustedURL(base)
	if err != nil {
		return nil, fmt.Errorf("SCROLL_LIFECYCLE_URL: %w", err)
	}
	authURL, err := trustedURL(auth)
	if err != nil || authURL.Scheme != coreURL.Scheme || authURL.Host != coreURL.Host {
		return nil, errors.New("SCROLL_AUTH_URL must share Core's trusted origin")
	}
	host := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(os.Getenv("SCROLL_REGISTRY_HOST"), "https://"), "http://"), "/")
	p := &publisher{base: base, auth: auth, owner: os.Getenv("SCROLL_OWNER_ID"), host: host,
		email: os.Getenv("SCROLL_PUBLISH_EMAIL"), password: os.Getenv("SCROLL_PUBLISH_PASSWORD"), manifest: os.Getenv("SCROLL_PUSH_MANIFEST")}
	if p.owner == "" || p.host == "" || strings.ContainsAny(p.host, "/\\") || p.email == "" || p.password == "" {
		return nil, errors.New("owner, registry host and ordinary publisher sign-in credentials are required")
	}
	jar, _ := cookiejar.New(nil)
	p.client = &http.Client{Timeout: 70 * time.Minute, Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return p, nil
}

func trustedURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "druid-gs"))) {
		return nil, errors.New("expected HTTPS URL (HTTP only for local development)")
	}
	return u, nil
}

func (p *publisher) authRequest(method, path string, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, p.auth+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("invalid authentication request")
	}
	req.Header.Set("Content-Type", "application/json")
	parsed, _ := url.Parse(p.auth)
	req.Header.Set("Origin", parsed.Scheme+"://"+parsed.Host)
	resp, err := p.client.Do(req)
	if err != nil {
		return errors.New("publisher authentication failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("publisher authentication rejected: HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
	}
	return nil
}

func (p *publisher) request(method, path string, body, out any) error {
	var jwt struct {
		Token string `json:"token"`
	}
	if err := p.authRequest(http.MethodGet, "/token", nil, &jwt); err != nil {
		return err
	}
	if jwt.Token == "" {
		return errors.New("authentication returned no owner token")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, p.base+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("invalid lifecycle request")
	}
	req.Header.Set("Authorization", "Bearer "+jwt.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return errors.New("lifecycle request failed; inspect published state before retrying")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("lifecycle request rejected: HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
	}
	return nil
}

func (p *publisher) preflight() error {
	var project struct {
		Registry string `json:"registry"`
		Project  string `json:"project"`
	}
	if err := p.request(http.MethodGet, "/v1/registry/project", nil, &project); err != nil {
		return err
	}
	if project.Registry != p.host || project.Project != p.host+"/"+p.owner {
		return errors.New("authenticated owner or registry differs from publisher configuration")
	}
	return nil
}

func (p *publisher) parseRef(ref string) (string, string, error) {
	prefix := p.host + "/"
	if !strings.HasPrefix(ref, prefix) {
		return "", "", errors.New("push manifest contains another registry")
	}
	project, remainder, ok := strings.Cut(strings.TrimPrefix(ref, prefix), "/")
	if !ok {
		return "", "", errors.New("push manifest contains no repository")
	}
	name, tag, ok := strings.Cut(remainder, ":")
	if !ok || !namePattern.MatchString(name) || !tagPattern.MatchString(tag) {
		return "", "", errors.New("push manifest contains an invalid tagged Scroll")
	}
	if project != p.owner {
		return "", "", errors.New("push manifest contains a repository outside the owner project")
	}
	return name, ref, nil
}

func (p *publisher) publishManifest() error {
	if os.Getenv("SCROLL_LIFECYCLE_REVIEWED") != "1" {
		return errors.New("public publication requires SCROLL_LIFECYCLE_REVIEWED=1")
	}
	if p.manifest == "" {
		return errors.New("SCROLL_PUSH_MANIFEST is required")
	}
	file, err := os.Open(p.manifest)
	if err != nil {
		return err
	}
	defer file.Close()
	selected := map[string]map[string]bool{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		name, ref, err := p.parseRef(scanner.Text())
		if err != nil {
			return err
		}
		if selected[name] == nil {
			selected[name] = map[string]bool{}
		}
		selected[name][ref] = true
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(selected) == 0 {
		return errors.New("no pushed Scroll artifacts to publish")
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var snapshot []publicationArtifact
		if err := p.request(http.MethodGet, "/v1/registry/repositories/"+url.PathEscape(name)+"/publication-preview", nil, &snapshot); err != nil {
			return fmt.Errorf("preview %s: %w", name, err)
		}
		if len(snapshot) == 0 {
			return fmt.Errorf("empty publication preview for %s", name)
		}
		previewTags := map[string]string{}
		for _, artifact := range snapshot {
			if !digestPattern.MatchString(artifact.Digest) || artifact.Tags == nil {
				return fmt.Errorf("invalid publication preview for %s", name)
			}
			for _, tag := range artifact.Tags {
				if !tagPattern.MatchString(tag) || previewTags[tag] != "" {
					return fmt.Errorf("invalid publication tag for %s", name)
				}
				previewTags[tag] = artifact.Digest
			}
		}
		refs := make([]string, 0, len(selected[name]))
		for ref := range selected[name] {
			refs = append(refs, ref)
		}
		sort.Strings(refs)
		for _, ref := range refs {
			var resolved struct {
				Repository string `json:"repository"`
				Digest     string `json:"digest"`
				Artifact   string `json:"artifact"`
			}
			if err := p.request(http.MethodGet, "/v1/registry/scrolls/revision?artifact="+url.QueryEscape(ref), nil, &resolved); err != nil {
				return fmt.Errorf("resolve %s: %w", ref, err)
			}
			repository := ref[:strings.LastIndex(ref, ":")]
			if resolved.Repository != repository || !digestPattern.MatchString(resolved.Digest) || resolved.Artifact != repository+"@"+resolved.Digest {
				return fmt.Errorf("resolved revision differs from pushed repository %s", repository)
			}
			if previewTags[ref[strings.LastIndex(ref, ":")+1:]] != resolved.Digest {
				return fmt.Errorf("pushed revision changed since publication preview: %s", ref)
			}
		}
		input := map[string]any{"repository": name, "expectedSnapshot": snapshot, "reviewedPublicContent": true}
		if err := p.request(http.MethodPost, "/v1/registry/scrolls/publish", input, nil); err != nil {
			return fmt.Errorf("publish %s: %w", name, err)
		}
		for _, ref := range refs {
			fmt.Println(ref)
		}
	}
	return nil
}

func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "preflight" && os.Args[1] != "publish") {
		return errors.New("usage: publish-owned preflight|publish")
	}
	p, err := configured()
	if err != nil {
		return err
	}
	if os.Getenv("SCROLL_LIFECYCLE_REVIEWED") != "1" {
		return errors.New("public publication requires reviewed content")
	}
	if err := p.authRequest(http.MethodPost, "/sign-in/email", map[string]string{"email": p.email, "password": p.password}, nil); err != nil {
		return err
	}
	defer p.authRequest(http.MethodPost, "/sign-out", map[string]any{}, nil)
	if err := p.preflight(); err != nil {
		return err
	}
	if os.Args[1] == "publish" {
		return p.publishManifest()
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
