package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/basicflag"
	"github.com/knadh/koanf/providers/env"
	"github.com/sirupsen/logrus"
	"github.com/thunderbottom/terrakube/pkg/kubeutils"
	"k8s.io/apimachinery/pkg/runtime"
)

// getLogger returns a new instance of logrus.Logger
// with LogLevel set to INFO
func getLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.InfoLevel)

	return logger
}

// readConfig parses the flags passed as arguments to the utility
// and fetches environment variables prefixed with `TK_` as config
func readConfig() (*koanf.Koanf, error) {
	var k = koanf.New(".")
	f := flag.NewFlagSet("terrakube", flag.ExitOnError)
	f.String("input", "STDIN", "Input directory/file(s) containing the Kubernetes YAML manifests.")
	f.String("output", "STDOUT", "Output file for the generated terraform configuration.")
	f.Bool("overwrite", false, "Overwrite existing terraform configuration file.")

	f.Parse(os.Args[1:])

	if err := k.Load(basicflag.Provider(f, "."), nil); err != nil {
		return nil, fmt.Errorf("Error loading configuration: %v", err)
	}

	k.Load(env.Provider("TK_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "TK_")), "_", "-", -1)
	}), nil)

	return k, nil
}

// parseManifests parses the input directory/file(s) for Kubernetes
// manifests and returns parsed YAMLs as manifest struct objects
func parseManifests(fp string) ([]runtime.Object, error) {
	var rto []runtime.Object
	if fp == "STDIN" || fp == "" {
		// check if stdin is available
		stat, err := os.Stdin.Stat()
		if err != nil {
			return nil, err
		}
		// check if stdin contains any data
		if stat.Mode()&os.ModeCharDevice != 0 || stat.Size() <= 0 {
			return nil, fmt.Errorf("No data received from STDIN.")
		}

		r := bufio.NewReader(os.Stdin)
		rto, err = kubeutils.Deserialize(r)
		if err != nil {
			return nil, err
		}
	} else {
		// passed filepath is either a file or a directory
		f, err := os.Stat(fp)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf(`Path "%v" does not exist.`, fp)
			}
			return nil, err
		}

		var fileList []string
		// check if passed filepath is a file or a directory
		switch mode := f.Mode(); {
		case mode.IsDir():
			fileList, err = findManifests(fp)
			if err != nil {
				return nil, err
			}
			if len(fileList) == 0 {
				return nil, fmt.Errorf(`Path "%v" does not contain any Kubernetes manifests.`, fp)
			}
		case mode.IsRegular():
			fileList = append(fileList, fp)
		}

		// read file, deserialize, and add to runtime objects slice
		for _, file := range fileList {
			f, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			r := bufio.NewReader(f)
			o, err := kubeutils.Deserialize(r)
			if err != nil {
				return nil, err
			}
			rto = append(rto, o...)
		}
	}
	return rto, nil
}

// findManifests recursively walks the passed directory and returns
// a slice containing the string path to Kubernetes YAML manifests
func findManifests(fp string) ([]string, error) {
	var fileList []string

	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := regexp.MatchString(".*.(yml|yaml)", filepath.Base(path)); err != nil {
			return err
		} else if matched {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileList, nil
}
