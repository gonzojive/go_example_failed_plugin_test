package codeloader

import (
	"context"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"

	"github.com/golang/glog"
)

var RegisteredPlugins []string

func RegisterPlugin(s string) {
	RegisteredPlugins = append(RegisteredPlugins, s)
}

type Config struct {
	goBinaryPath string
}

func DefaultConfig(ctx context.Context) (*Config, error) {
	c := &Config{goBinaryPath: "go"}
	if err := c.ensureCanCompilePluginCode(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (config *Config) Version(ctx context.Context) (string, error) {
	bytes, err := exec.CommandContext(ctx, config.goBinaryPath, "version").Output()
	return string(bytes), err
}

func (config *Config) ensureCanCompilePluginCode(ctx context.Context) error {
	wantVersion := fmt.Sprintf("go version %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	gotVersion, err := config.Version(ctx)
	gotVersion = strings.TrimSpace(gotVersion)
	if err != nil {
		return fmt.Errorf("failed to get go version: %w", err)
	}
	if gotVersion != wantVersion {
		return fmt.Errorf("compiler version used for compiling plugins (%q) must match the version of the compiler runtime (%q)", gotVersion, wantVersion)
	}
	return nil
}

type Request struct {
	// Go source code with no imports.
	SourceCode string
	Files      map[string]string
	Config     *Config
}

type Response struct {
	// Path that may be passed to plugin.Open to load the plugin.
	Path   string
	Plugin *plugin.Plugin
}

func CompileAndLoadCompileTimeCode(req *Request) (*Response, error) {
	glog.Infof("build context: %+v", build.Default)

	//bi, ok := debug.ReadBuildInfo()
	// if !ok {
	// 	return nil, fmt.Errorf("failed to read build info.. compiler not built with module support; build context: %+v", build.Default)
	// }
	//glog.Infof("build info: %+v", bi)
	workingDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		return nil, err
	}
	//defer os.RemoveAll(workingDir) // clean up

	if err := ioutil.WriteFile(filepath.Join(workingDir, "theplugin.go"), []byte(req.SourceCode), 0664); err != nil {
		return nil, err
	}
	for p, contents := range req.Files {
		if err := ioutil.WriteFile(filepath.Join(workingDir, p), []byte(contents), 0664); err != nil {
			return nil, err
		}
	}

	pluginPath := filepath.Join(workingDir, "theplugin.so")

	cmd := exec.Command(req.Config.goBinaryPath, "build", "-buildmode=plugin", "theplugin.go")
	cmd.Dir = workingDir

	stdOut, stdErr := strings.Builder{}, strings.Builder{}
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("problem compiling plugin from directory %q: %v\n  %s\n  %s", workingDir, err, stdOut.String(), stdErr.String())
	}
	glog.Infof("plugin compilation succeeded: %q %q - output at %q", stdOut.String(), stdErr.String(), workingDir)

	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open compiled plugin %q: %w", pluginPath, err)
	}

	return &Response{
		Path:   pluginPath,
		Plugin: p,
	}, nil
}
