package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"github.com/highcard-dev/scrolls-registry/internal/registry"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func RecreateDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func TarDirectory(srcPath string, destinationPath string) error {
	var buf bytes.Buffer
	zr := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zr)
	filepath.Walk(srcPath, func(file string, fi os.FileInfo, err error) error {
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}
		fileName := strings.TrimPrefix(strings.TrimPrefix(file, srcPath), strings.TrimPrefix(srcPath, "./"))
		if fileName == "" {
			header.Name = "./"
		} else {
			header.Name = "." + filepath.ToSlash(fileName)
		}

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})

	if err := tw.Close(); err != nil {
		return err
	}
	if err := zr.Close(); err != nil {
		return err
	}
	fileToWrite, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE, os.FileMode(777))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fileToWrite, &buf); err != nil {
		return err
	}
	return nil
}

func CreateYamlFile(data registry.Registry, destinationPath string) error {
	yamlData, err := yaml.Marshal(&data)

	if err != nil {
		return err
	}
	err = os.WriteFile(destinationPath, yamlData, 0644)
	if err != nil {
		return err
	}
	return nil
}
