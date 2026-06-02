package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	cp "github.com/otiai10/copy"
	"gopkg.in/yaml.v3"
)

type prebuildSpec struct {
	Target   string
	Artifact string
	Source   string
	Image    string
	Ports    []string
	MinDisk  string
	MinRAM   string
	MinCPU   string
	Category string
	Smart    bool
	PackMeta bool
}

type scrollFile struct {
	Commands map[string]command `yaml:"commands"`
}

type command struct {
	Procedures []procedure `yaml:"procedures"`
}

type procedure struct {
	Image      string            `yaml:"image"`
	Mounts     []mount           `yaml:"mounts"`
	WorkingDir string            `yaml:"working_dir"`
	Env        map[string]string `yaml:"env"`
	Command    []string          `yaml:"command"`
}

type mount struct {
	Path    string `yaml:"path"`
	SubPath string `yaml:"sub_path"`
}

func main() {
	targets := flag.String("targets", "all-steam", "one target, comma-separated targets, or all-steam")
	flag.Parse()

	specs, err := selectSpecs(*targets)
	if err != nil {
		fail(err)
	}
	for _, spec := range specs {
		if err := runSpec(spec); err != nil {
			fail(fmt.Errorf("%s: %w", spec.Target, err))
		}
	}
}

func runSpec(spec prebuildSpec) error {
	root, err := createPrebuildRoot(spec.Target)
	if err != nil {
		return err
	}
	if strings.EqualFold(os.Getenv("PREBUILD_KEEP_ROOT"), "true") {
		fmt.Printf("PREBUILD_KEEP_ROOT=true, preserving artifact root at %s\n", root)
	} else {
		defer os.RemoveAll(root)
	}

	if err := copyScrollSource(spec.Source, root); err != nil {
		return fmt.Errorf("copy scroll source: %w", err)
	}

	scroll, err := loadScroll(filepath.Join(root, "scroll.yaml"))
	if err != nil {
		return err
	}
	install, ok := scroll.Commands["install"]
	if !ok {
		return errors.New("scroll has no install command")
	}

	mounts := newDockerMountSet(root, spec.Target)
	defer mounts.cleanup()

	fmt.Printf("Prebuilding %s from %s\n", spec.Artifact, spec.Source)
	for index, proc := range install.Procedures {
		if err := runProcedure(root, spec, mounts, index, proc); err != nil {
			_ = mounts.copyBack()
			return err
		}
	}
	if err := mounts.copyBack(); err != nil {
		return err
	}
	if err := validateInstalledRoot(root); err != nil {
		return err
	}
	if strings.EqualFold(os.Getenv("PREBUILD_PUSH"), "false") {
		fmt.Printf("PREBUILD_PUSH=false, leaving verified artifact root at %s\n", root)
		return nil
	}
	return pushArtifact(root, spec)
}

func createPrebuildRoot(target string) (string, error) {
	parent := os.Getenv("PREBUILD_TMP_PARENT")
	if parent == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		parent = filepath.Join(cwd, ".prebuild-tmp")
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(parent, "druid-prebuild-"+target+"-")
}

func copyScrollSource(source, root string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := cp.Copy(filepath.Join(source, entry.Name()), filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func runProcedure(root string, spec prebuildSpec, mounts *dockerMountSet, index int, proc procedure) error {
	if proc.Image == "" {
		proc.Image = spec.Image
	}
	if len(proc.Command) == 0 {
		return fmt.Errorf("install procedure %d has no command", index)
	}

	args := []string{"run", "--rm"}
	if platform := os.Getenv("PREBUILD_DOCKER_PLATFORM"); platform != "" {
		args = append(args, "--platform", platform)
	}
	for _, m := range proc.Mounts {
		volumeName, err := mounts.volumeFor(proc.Image, m)
		if err != nil {
			return err
		}
		args = append(args, "-v", volumeName+":"+m.Path)
	}
	for key, value := range proc.Env {
		args = append(args, "-e", key+"="+value)
	}
	args = append(args,
		"-e", "DRUID_RUNTIME_BACKEND=docker",
		"-e", "DRUID_ROOT=/scroll",
	)
	if proc.WorkingDir != "" {
		args = append(args, "-w", proc.WorkingDir)
	}
	args = append(args, "--entrypoint", proc.Command[0], proc.Image)
	args = append(args, proc.Command[1:]...)

	fmt.Printf("Running install procedure %d with %s\n", index, proc.Image)
	return run("docker", args...)
}

func mountHostPath(root string, m mount) string {
	if m.SubPath == "." {
		return root
	}
	subPath := strings.Trim(m.SubPath, "/")
	if subPath == "" {
		subPath = "data"
	}
	return filepath.Join(root, subPath)
}

type dockerMountSet struct {
	root    string
	target  string
	volumes map[string]dockerMountVolume
}

type dockerMountVolume struct {
	HostPath string
	Name     string
	Image    string
}

func newDockerMountSet(root, target string) *dockerMountSet {
	return &dockerMountSet{
		root:    root,
		target:  target,
		volumes: map[string]dockerMountVolume{},
	}
}

func (m *dockerMountSet) volumeFor(image string, mount mount) (string, error) {
	hostPath := mountHostPath(m.root, mount)
	if existing, ok := m.volumes[hostPath]; ok {
		return existing.Name, nil
	}
	if err := os.MkdirAll(hostPath, 0o755); err != nil {
		return "", err
	}

	name := fmt.Sprintf("druid-prebuild-%s-%d-%d", sanitizeName(m.target), len(m.volumes), os.Getpid())
	if err := run("docker", "volume", "create", name); err != nil {
		return "", err
	}
	volume := dockerMountVolume{HostPath: hostPath, Name: name, Image: image}
	m.volumes[hostPath] = volume

	helper := name + "-seed"
	removeContainer(helper)
	if err := run("docker", helperCreateArgs(helper, image, name)...); err != nil {
		return "", err
	}
	if err := run("docker", "cp", hostPath+"/.", helper+":/volume"); err != nil {
		removeContainer(helper)
		return "", err
	}
	if err := removeContainer(helper); err != nil {
		return "", err
	}

	chownArgs := []string{"run", "--rm"}
	if platform := os.Getenv("PREBUILD_DOCKER_PLATFORM"); platform != "" {
		chownArgs = append(chownArgs, "--platform", platform)
	}
	chownArgs = append(chownArgs, "--user", "root", "-v", name+":/volume", "--entrypoint", "chown", image, "-R", "1000:1000", "/volume")
	if err := run("docker", chownArgs...); err != nil {
		return "", err
	}
	return name, nil
}

func (m *dockerMountSet) copyBack() error {
	for _, volume := range m.volumes {
		if err := copyVolumeToHost(volume); err != nil {
			return err
		}
	}
	return nil
}

func (m *dockerMountSet) cleanup() {
	for _, volume := range m.volumes {
		_ = run("docker", "volume", "rm", "-f", volume.Name)
	}
}

func copyVolumeToHost(volume dockerMountVolume) error {
	helper := volume.Name + "-copy"
	removeContainer(helper)
	if err := run("docker", helperCreateArgs(helper, volume.Image, volume.Name)...); err != nil {
		return err
	}
	if err := os.RemoveAll(volume.HostPath); err != nil {
		removeContainer(helper)
		return err
	}
	if err := os.MkdirAll(volume.HostPath, 0o755); err != nil {
		removeContainer(helper)
		return err
	}
	if err := run("docker", "cp", helper+":/volume/.", volume.HostPath); err != nil {
		removeContainer(helper)
		return err
	}
	return removeContainer(helper)
}

func helperCreateArgs(name, image, volume string) []string {
	args := []string{"create", "--name", name}
	if platform := os.Getenv("PREBUILD_DOCKER_PLATFORM"); platform != "" {
		args = append(args, "--platform", platform)
	}
	return append(args, "-v", volume+":/volume", "--entrypoint", "true", image)
}

func removeContainer(name string) error {
	cmd := exec.Command("docker", "rm", "-f", name)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func sanitizeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	return b.String()
}

func pushArtifact(root string, spec prebuildSpec) error {
	loginArgs := []string{
		"login",
		"--host", os.Getenv("SCROLL_REGISTRY_HOST"),
		"--user", os.Getenv("SCROLL_REGISTRY_USER"),
		"--password", os.Getenv("SCROLL_REGISTRY_PASSWORD"),
	}
	if err := run("druid", loginArgs...); err != nil {
		return err
	}

	args := []string{"push", spec.Artifact, root, "-i", spec.Image}
	for _, port := range spec.Ports {
		args = append(args, "-p", port)
	}
	if spec.PackMeta {
		args = append(args, "-m")
	}
	if spec.MinDisk != "" {
		args = append(args, "--min-disk", spec.MinDisk)
	}
	if spec.MinRAM != "" {
		args = append(args, "--min-ram", spec.MinRAM)
	}
	if spec.MinCPU != "" {
		args = append(args, "--min-cpu", spec.MinCPU)
	}
	if spec.Smart {
		args = append(args, "--smart")
	}
	if spec.Category != "" {
		args = append(args, "--category", spec.Category)
	}
	return run("druid", args...)
}

func validateInstalledRoot(root string) error {
	if _, err := os.Stat(filepath.Join(root, "scroll.yaml")); err != nil {
		return fmt.Errorf("missing scroll.yaml after prebuild: %w", err)
	}
	dataRoot := filepath.Join(root, "data")
	info, err := os.Stat(dataRoot)
	if err != nil {
		return fmt.Errorf("missing data directory after prebuild: %w", err)
	}
	if !info.IsDir() {
		return errors.New("data is not a directory")
	}
	var regularFiles int
	if err := filepath.WalkDir(dataRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			regularFiles++
		}
		return nil
	}); err != nil {
		return err
	}
	if regularFiles == 0 {
		return errors.New("data directory is empty after prebuild")
	}
	return nil
}

func loadScroll(path string) (*scrollFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scroll scrollFile
	if err := yaml.Unmarshal(content, &scroll); err != nil {
		return nil, err
	}
	return &scroll, nil
}

func selectSpecs(raw string) ([]prebuildSpec, error) {
	all := allSpecs()
	if raw == "" || raw == "all-steam" {
		return all, nil
	}
	byTarget := map[string]prebuildSpec{}
	for _, spec := range all {
		byTarget[spec.Target] = spec
		byTarget[strings.TrimSuffix(spec.Target, "server")] = spec
		byTarget[spec.Artifact] = spec
	}
	var selected []prebuildSpec
	for _, target := range strings.Split(raw, ",") {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		spec, ok := byTarget[target]
		if !ok {
			return nil, fmt.Errorf("unknown prebuild target %q", target)
		}
		selected = append(selected, spec)
	}
	return selected, nil
}

func allSpecs() []prebuildSpec {
	steamImage := getenv("DRUID_STEAM_RUNTIME_IMAGE", "artifacts.druid.gg/druid-team/druid:v0.1.243-steamcmd")
	specs := []prebuildSpec{
		{Target: "pwserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:pwserver-prebuild", Source: "./scrolls/lgsm/pwserver", Image: steamImage, Ports: []string{"main=8211/udp", "rcon=25575"}, MinDisk: "7Gi", MinRAM: "2Gi", MinCPU: "0.5", Category: "palworld", Smart: true, PackMeta: true},
		{Target: "arkserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:arkserver-prebuild", Source: "./scrolls/lgsm/arkserver", Image: steamImage, Ports: []string{"main=/udp", "query=/udp", "rcon"}, MinDisk: "25Gi", MinRAM: "7Gi", MinCPU: "0.5", Category: "ark", Smart: true, PackMeta: true},
		{Target: "untserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:untserver-prebuild", Source: "./scrolls/lgsm/untserver", Image: steamImage, Ports: []string{"main=/udp"}, MinDisk: "7Gi", MinRAM: "1Gi", MinCPU: "0.5", Category: "unturned", Smart: true, PackMeta: true},
		{Target: "dayzserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:dayzserver-prebuild", Source: "./scrolls/lgsm/dayzserver", Image: steamImage, Ports: []string{"main=/udp", "battle-eye=2304/udp", "query=27016/udp"}, MinDisk: "7Gi", MinRAM: "5Gi", MinCPU: "1", Category: "dayz", PackMeta: true},
		{Target: "sdtdserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:sdtdserver-prebuild", Source: "./scrolls/lgsm/sdtdserver", Image: steamImage, Ports: []string{"main=26900/udp", "main2=26902/udp", "maintcp=26900"}, MinDisk: "20Gi", MinRAM: "2Gi", MinCPU: "0.5", Category: "7days", PackMeta: true},
		{Target: "gmodserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:gmodserver-prebuild", Source: "./scrolls/lgsm/gmodserver", Image: steamImage, Ports: []string{"query=27005/udp", "main=/udp", "sourcetv=27020/udp", "steam=27015"}, MinDisk: "8Gi", MinRAM: "512Mi", MinCPU: "0.25", Category: "gmod", Smart: true, PackMeta: true},
		{Target: "cs2server", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:cs2server-prebuild", Source: "./scrolls/lgsm/cs2server", Image: steamImage, Ports: []string{"main=/udp", "rcon=27015"}, MinDisk: "38Gi", MinRAM: "1Gi", MinCPU: "0.5", Category: "cs2", Smart: true, PackMeta: true},
		{Target: "pzserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:pzserver-prebuild", Source: "./scrolls/lgsm/pzserver", Image: steamImage, Ports: []string{"main=/udp", "main2=/udp"}, MinDisk: "3Gi", MinRAM: "512Mi", MinCPU: "0.25", Category: "zomboid", Smart: true, PackMeta: true},
		{Target: "csgoserver", Artifact: "artifacts.druid.gg/druid-team/scroll-lgsm:csgoserver-prebuild", Source: "./scrolls/lgsm/csgoserver", Image: steamImage, Ports: []string{"query=27005/udp", "main=27015/udp", "sourcetv=27020/udp", "steam=27015"}, Category: "csgo", Smart: true, PackMeta: true},
		{Target: "rust-vanilla", Artifact: "artifacts.druid.gg/druid-team/scroll-rust-vanilla:latest-prebuild", Source: "./scrolls/rust/rust-vanilla/latest", Image: steamImage, Ports: []string{"main=/udp", "query=/udp", "rcon", "rustplus"}, MinDisk: "10Gi", MinRAM: "6Gi", MinCPU: "1", Category: "rust", Smart: true},
		{Target: "rust-oxide", Artifact: "artifacts.druid.gg/druid-team/scroll-rust-oxide:latest-prebuild", Source: "./scrolls/rust/rust-oxide/latest", Image: steamImage, Ports: []string{"main=/udp", "query=/udp", "rcon", "rustplus"}, MinDisk: "10Gi", MinRAM: "6Gi", MinCPU: "1", Category: "rust", Smart: true},
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Target < specs[j].Target })
	return specs
}

func run(name string, args ...string) error {
	fmt.Printf("+ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
