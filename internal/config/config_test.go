// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// envOf returns an environment reader over a map.
func envOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestDir(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{
			name: "linux with an XDG home",
			goos: "linux",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg"},
			want: filepath.Join("/xdg", "swag"),
		},
		{
			name: "linux without an XDG home",
			goos: "linux",
			env:  map[string]string{"HOME": "/home/user"},
			want: filepath.Join("/home/user", ".config", "swag"),
		},
		{
			name: "macOS",
			goos: "darwin",
			env:  map[string]string{"HOME": "/Users/user"},
			want: filepath.Join("/Users/user", "Library", "Application Support", "swag"),
		},
		{
			name: "windows",
			goos: "windows",
			env:  map[string]string{"AppData": filepath.Join("C:", "Users", "user", "AppData", "Roaming")},
			want: filepath.Join("C:", "Users", "user", "AppData", "Roaming", "swag"),
		},
		{
			name: "a directory override wins",
			goos: "windows",
			env:  map[string]string{"SWAG_CONFIG_DIR": "/portable/swag/", "AppData": "ignored"},
			want: filepath.Clean("/portable/swag/"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Dir(tt.goos, envOf(tt.env))
			if err != nil {
				t.Fatalf("Dir: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Dir(%s) = %q, want %q", tt.goos, got, tt.want)
			}
		})
	}
}

func TestDirErrors(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{"linux without a home", "linux", map[string]string{}, "XDG_CONFIG_HOME"},
		{"macOS without a home", "darwin", map[string]string{}, "HOME"},
		{"windows without AppData", "windows", map[string]string{}, "AppData"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dir(tt.goos, envOf(tt.env))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Dir error = %v, want a mention of %q", err, tt.want)
			}
		})
	}
}

func TestFile(t *testing.T) {
	got, err := File("linux", envOf(map[string]string{"XDG_CONFIG_HOME": "/xdg"}))
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if want := filepath.Join("/xdg", "swag", FileName); got != want {
		t.Fatalf("File = %q, want %q", got, want)
	}

	got, err = File("linux", envOf(map[string]string{"SWAG_CONFIG": "/portable/swag.toml", "XDG_CONFIG_HOME": "ignored"}))
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if got != "/portable/swag.toml" {
		t.Fatalf("a file override = %q, want the named file", got)
	}
}

func TestFileError(t *testing.T) {
	if _, err := File("linux", envOf(map[string]string{})); err == nil {
		t.Fatal("a platform with no home must fail the file path")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(file, []byte("# swag\n"), 0o644); err != nil {
		t.Fatalf("write the file: %v", err)
	}
	if !Exists(file) {
		t.Error("a present file must report true")
	}
	if Exists(filepath.Join(dir, "missing.toml")) {
		t.Error("a missing file must report false")
	}
	if Exists(dir) {
		t.Error("a directory is not a configuration file")
	}
}
