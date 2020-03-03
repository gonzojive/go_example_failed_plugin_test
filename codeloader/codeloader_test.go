package codeloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gonzojive/go_example_failed_plugin_test/interactionok"
)

//const absPackage = "/home/red/git/go-example-plugin-test-failure"

func TestConfig_ensureCanCompilePluginCode(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name          string
		configCreator func() (*Config, error)
		wantErr       bool
	}{
		{"default context", func() (*Config, error) { return DefaultConfig(ctx) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.configCreator()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Config creation error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := config.ensureCanCompilePluginCode(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Config.ensureCanCompilePluginCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_CompileAndLoadCompileTimeCode(t *testing.T) {
	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get PWD: %v", err)
	}
	if _, err := os.Stat(filepath.Join(testDir, "codeloader_test.go")); err != nil {
		t.Fatalf("must run go test from root of example project... does not appear that working dir is correct (%v)", err)
	}
	absPackageDir := filepath.Dir(testDir)
	fmt.Printf("will reference project root %q when compiling plugins", absPackageDir)

	ctx := context.Background()
	config, err := DefaultConfig(ctx)
	if err != nil {
		t.Fatalf("bad compiler comfig: %v", err)
	}
	tests := []struct {
		name          string
		program       string
		extraFiles    map[string]string
		afterLoadTest func(r *Response, t *testing.T)
	}{
		{
			"test that plugin compilation is working",
			`
			package main

			import (
				"fmt"
			)

			func init() {
				fmt.Printf("The code was loaded!")
			}

			var Value  = "lucy in the sky"
			`,
			nil,
			func(resp *Response, t *testing.T) {
				v, err := resp.Plugin.Lookup("Value")
				if err != nil {
					t.Fatalf("failed to lookup Value in plugin: %v", err)
				}
				got, ok := v.(*string)
				if !ok {
					t.Fatalf("Value is not a string: %v", v)
				}
				if want := "lucy in the sky"; *got != want {
					t.Fatalf("value unexpected: got %v, want %v", got, want)
				}
			},
		},
		{
			"show example of interaction of a plugin with a non-test package",
			`
			package main

			import (
				"github.com/gonzojive/go_example_failed_plugin_test/interactionok"
			)

			func init() {
				interactionok.RegisterPlugin("interaction confirmed")
			}
			`,
			map[string]string{
				"go.mod": fmt.Sprintf(`module example.com/me/hello1

				require (
					github.com/gonzojive/go_example_failed_plugin_test v0.0.0
				)

				replace github.com/gonzojive/go_example_failed_plugin_test => %s
				`, absPackageDir),
			},
			func(resp *Response, t *testing.T) {
				if got, want := interactionok.RegisteredPlugins, []string{"interaction confirmed"}; !reflect.DeepEqual(got, want) {
					t.Fatalf("manipulates RegisteredPlugins: got %q, want %q", got, want)
				}
			},
		},
		{
			"show example of interaction of a plugin with a non-test package",
			`
			package main

			import (
				"github.com/gonzojive/go_example_failed_plugin_test/codeloader"
			)

			func init() {
				codeloader.RegisterPlugin("interaction confirmed")
			}
			`,
			map[string]string{
				"go.mod": fmt.Sprintf(`module example.com/me/hello2

				require (
					github.com/gonzojive/go_example_failed_plugin_test v0.0.0
				)

				replace github.com/gonzojive/go_example_failed_plugin_test => %s
				`, absPackageDir),
			},
			func(resp *Response, t *testing.T) {
				if got, want := RegisteredPlugins, []string{"interaction confirmed"}; !reflect.DeepEqual(got, want) {
					t.Fatalf("manipulates RegisteredPlugins: got %q, want %q", got, want)
				}
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resp, err := CompileAndLoadCompileTimeCode(&Request{
				Config:     config,
				SourceCode: tt.program,
				Files:      tt.extraFiles,
			})
			if err != nil {
				t.Fatalf("CompileAndLoadCompileTimeCode() returned error: %v", err)
			}
			if resp == nil {
				t.Fatalf("got nil response")
			}
			tt.afterLoadTest(resp, t)
		})
	}
}
