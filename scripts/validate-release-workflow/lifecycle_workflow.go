package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type lifecycleWorkflow struct {
	Jobs map[string]lifecycleJob `yaml:"jobs"`
}

type lifecycleJob struct {
	Uses        string            `yaml:"uses"`
	With        map[string]any    `yaml:"with"`
	Secrets     any               `yaml:"secrets"`
	Env         map[string]string `yaml:"env"`
	Needs       string            `yaml:"needs"`
	If          string            `yaml:"if"`
	RunsOn      string            `yaml:"runs-on"`
	Environment string            `yaml:"environment"`
	Concurrency struct {
		Cancel bool `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`
	Steps []lifecycleStep `yaml:"steps"`
}

type lifecycleStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
	With map[string]any    `yaml:"with"`
}

// Check the actual CI callers, not only the opt-in helper. Missing credentials
// must never silently select the legacy direct-publication path.
func validateLifecycleWorkflows(directory string) error {
	read := func(name string) (lifecycleWorkflow, error) {
		var workflow lifecycleWorkflow
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err == nil {
			err = yaml.Unmarshal(data, &workflow)
		}
		return workflow, err
	}
	for file, public := range map[string]bool{"release.yml": true, "pr.yml": false} {
		workflow, err := read(file)
		if err != nil {
			return err
		}
		job := workflow.Jobs["build-deploy"]
		if len(workflow.Jobs) != 1 || job.Uses != "./.github/workflows/scroll-lifecycle.yml" || job.With["public"] != public || job.Secrets != nil {
			return fmt.Errorf("%s must call the shared lifecycle with public=%t and no inherited secrets", file, public)
		}
	}
	workflow, err := read("scroll-lifecycle.yml")
	if err != nil {
		return err
	}
	verify, publish := workflow.Jobs["verify"], workflow.Jobs["publish"]
	if publish.Needs != "verify" || publish.Environment != "${{ inputs.public && 'scroll-release' || 'scroll-preview' }}" || publish.RunsOn != "ubuntu-latest" || publish.Concurrency.Cancel {
		return fmt.Errorf("publication requires verification, protected environments, a fresh runner and non-cancelling concurrency")
	}
	wantCondition := "needs.verify.outputs.changed == 'true' && ((inputs.public && github.event_name != 'pull_request') || (!inputs.public && github.event_name == 'pull_request' && github.event.pull_request.head.repo.full_name == github.repository))"
	if strings.Join(strings.Fields(publish.If), " ") != wantCondition {
		return fmt.Errorf("publication must reject fork PRs and public PR publication")
	}
	if len(verify.Env) != 0 || len(publish.Env) != 0 {
		return fmt.Errorf("lifecycle credentials/configuration must be scoped to steps, not whole jobs")
	}
	tests, gate, cli, publishCount := false, false, false, 0
	for _, step := range verify.Steps {
		if strings.Contains(step.Run, "go test ./scripts/publish-lifecycle ./scripts/stage-scroll-ui ./scripts/validate-release-workflow") {
			tests = true
		}
		for _, value := range step.Env {
			if strings.Contains(value, "secrets.") {
				return fmt.Errorf("validation must not receive publication secrets")
			}
		}
	}
	for index, step := range publish.Steps {
		if step.Name == "Require provisioned lifecycle environment" && step.Env["LIFECYCLE_READY"] == "${{ vars.SCROLL_LIFECYCLE_READY }}" && strings.Contains(step.Run, "exit 1") {
			if index != 0 {
				return fmt.Errorf("provisioning gate must precede every publication job step")
			}
			gate = true
		}
		if step.With["repository"] == "highcard-dev/druid-cli" {
			ref, _ := step.With["ref"].(string)
			cli = regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(ref) && step.With["persist-credentials"] == false
		}
		if step.Run != "bash ./scripts/push.sh" {
			continue
		}
		publishCount++
		for name, want := range map[string]string{
			"SCROLL_PUBLISH_MODE":                "lifecycle",
			"SCROLL_REGISTRY_NAMESPACE":          "${{ vars.SCROLL_LIFECYCLE_OWNER }}",
			"SCROLL_REGISTRY_RUNTIME_NAMESPACE":  "druid-team",
			"SCROLL_REGISTRY_USER":               "${{ secrets.SCROLL_TEAM_REGISTRY_USER }}",
			"SCROLL_REGISTRY_PASSWORD":           "${{ secrets.SCROLL_TEAM_REGISTRY_PASSWORD }}",
			"SCROLL_TEAM_EMAIL":                  "${{ secrets.SCROLL_TEAM_EMAIL }}",
			"SCROLL_TEAM_PASSWORD":               "${{ secrets.SCROLL_TEAM_PASSWORD }}",
			"SCROLL_TAG_SUFFIX":                  "-${{ github.event.pull_request.head.sha || github.sha }}",
			"SCROLL_LIFECYCLE_REPOSITORY_SUFFIX": "${{ github.event_name == 'pull_request' && format('-pr{0}', github.event.pull_request.number) || '' }}",
			"SCROLL_LIFECYCLE_PUBLISH":           "${{ inputs.public && '1' || '0' }}",
			"SCROLL_LIFECYCLE_REVIEWED":          "${{ inputs.public && '1' || '0' }}",
		} {
			if step.Env[name] != want {
				return fmt.Errorf("lifecycle publication has incorrect %s", name)
			}
		}
	}
	if !tests || !gate || !cli || publishCount != 1 {
		return fmt.Errorf("shared workflow needs publisher tests, provisioning gate, pinned CLI and one lifecycle catalog call")
	}
	return nil
}
