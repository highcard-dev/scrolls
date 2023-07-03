package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	cp "github.com/otiai10/copy"

	"github.com/blang/semver"
)

type TemplateVars struct {
	Artifact       string
	Version        string
	VersionEscaped string
	Artifacts      map[string]string
}

func replace(input, from, to string) string {
	return strings.Replace(input, from, to, -1)
}

func main() {
	println("The world needs to wake up from the darkness of ignorance.")

	//	funcMap := template.FuncMap{
	//		"replace": replace,
	//	}
	basepath := os.Args[1]

	if basepath == "" {
		fmt.Println("Please enter a path.")
		return
	}

	path := basepath + "/.build"
	artifactsPath := path + "/artifacts.json"
	switchScrollDir := path + "/scroll-switch"
	scrollYamlTemplate := path + "/scroll.yaml.tmpl"

	println("Path: " + path)
	println("Artifacts Path: " + artifactsPath)
	println("Switch Scroll Dir: " + switchScrollDir)
	println("Scroll Yaml Template: " + scrollYamlTemplate)

	//parse artifacs json file
	var artifacts map[string]string
	artifacts = make(map[string]string)
	artifactBytes, err := ioutil.ReadFile(artifactsPath)
	if err != nil {
		fmt.Printf("Error reading artifacts.json file. %s", err.Error())
		return
	}
	json.Unmarshal(artifactBytes, &artifacts)

	scollYamltemplate, err := template.ParseFiles(scrollYamlTemplate)
	// Capture any error
	if err != nil {
		log.Fatalln(err)
	}

	//iterate through artifacts and generate scroll.yaml files
	for version, artifact := range artifacts {
		println("Generating scroll.yaml for " + artifact + " version " + version)
		var templateVars TemplateVars
		templateVars.Artifact = artifact
		templateVars.Version = version
		templateVars.VersionEscaped = strings.Replace(version, ".", "-", -1)
		templateVars.Artifacts = GetArtifactsAbove(version, artifacts, true)

		//create scroll dir
		dir := filepath.Join(basepath, version)
		os.MkdirAll(dir, os.ModePerm)

		// Create a new file for the rendered output
		outputFile, err := os.Create(filepath.Join(dir, "scroll.yaml"))
		if err != nil {
			log.Println("create file: ", err)
			return
		}
		defer outputFile.Close()

		// Print out the template to std
		scollYamltemplate.Execute(outputFile, templateVars)

		cp.Copy(path+"/init-files", dir+"/init-files")
		cp.Copy(path+"/init-files-template", dir+"/init-files-template")
		cp.Copy(path+"/update", dir+"/update")

		//render and copy switch scroll
		subitems, _ := ioutil.ReadDir(switchScrollDir)
		for _, subitem := range subitems {
			if !subitem.IsDir() {
				continue
			}
			filename := subitem.Name()
			switchSrollYaml := filepath.Join(switchScrollDir, filename, "scroll-switch.sh.tmpl")

			drdScrollName := strings.TrimSuffix(filename, "@version")
			println("Generating switch scroll for " + drdScrollName + " version " + version)

			filecontent, err := ioutil.ReadFile(switchSrollYaml)
			if err != nil {
				log.Fatalln(err)
			}

			for version, artifact := range GetArtifactsAbove(version, artifacts, false) {

				escapedVersion := strings.Replace(version, ".", "-", -1)

				switchScrollDir := filepath.Join(dir, "scroll-switch", drdScrollName+"@"+escapedVersion)

				os.MkdirAll(switchScrollDir, os.ModePerm)

				RenderFile(switchScrollDir, "scroll-switch.sh", string(filecontent), artifact, version)
			}

		}

	}

}

func RenderFile(dir string, filename string, filecontent string, artifact string, version string) {

	escapedVersion := strings.Replace(version, ".", "-", -1)
	// Create a new file for the rendered output
	outputFile, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		log.Println("create file: ", err)
		return
	}
	defer outputFile.Close()

	templ, err := template.New(dir).Parse(string(filecontent))
	// Capture any error
	if err != nil {
		log.Fatalln(err)
	}
	templateVars := TemplateVars{Artifact: artifact, Version: version, VersionEscaped: escapedVersion}
	// Print out scroll switch template
	templ.Execute(outputFile, templateVars)
}

func GetArtifactsAbove(version string, artifacts map[string]string, escaped bool) map[string]string {

	artifactsAbove := make(map[string]string)
	for artifactVersion, artifact := range artifacts {

		aV := SafeParseVersion(artifactVersion)
		v := SafeParseVersion(version)
		fmt.Printf("aV %v v: %v \n", aV, v)

		if escaped {
			artifactVersion = strings.Replace(artifactVersion, ".", "-", -1)
		}

		if aV.GT(v) {
			artifactsAbove[artifactVersion] = artifact
		}
	}
	return artifactsAbove
}

func SafeParseVersion(version string) semver.Version {
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		log.Fatalln("Invalid version format " + version)
	}
	if len(parts) == 2 {
		version = version + ".0"
	}

	return semver.MustParse(version)

}
