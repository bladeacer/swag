// SPDX-License-Identifier: Apache-2.0

// Package config resolves the location of the swag configuration file.
//
// The location follows the convention of the platform:
//
//   - Linux and the other Unix systems: $XDG_CONFIG_HOME/swag, or
//     $HOME/.config/swag when the variable is empty
//   - macOS: $HOME/Library/Application Support/swag
//   - Windows: %AppData%\swag
//
// The SWAG_CONFIG_DIR environment variable names the directory outright and
// SWAG_CONFIG names the file, so a portable install needs no platform rule.
//
// The file format lands with v0.9.0. This package answers where the file
// lives, so every command looks in one place.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvDir names the environment variable that overrides the directory.
const EnvDir = "SWAG_CONFIG_DIR"

// EnvFile names the environment variable that overrides the file path.
const EnvFile = "SWAG_CONFIG"

// FileName is the name of the configuration file inside the directory.
const FileName = "config.toml"

// dirName is the directory name of the tool inside the platform
// configuration directory.
const dirName = "swag"

// Dir returns the configuration directory of the platform. The goos value
// selects the convention and env reads the environment, so a caller passes
// runtime.GOOS and os.Getenv and a test covers every platform.
func Dir(goos string, env func(string) string) (string, error) {
	if dir := env(EnvDir); dir != "" {
		return filepath.Clean(dir), nil
	}
	switch goos {
	case "windows":
		appData := env("AppData")
		if appData == "" {
			return "", fmt.Errorf("config: %%AppData%% is not defined")
		}
		return filepath.Join(appData, dirName), nil
	case "darwin":
		home := env("HOME")
		if home == "" {
			return "", fmt.Errorf("config: $HOME is not defined")
		}
		return filepath.Join(home, "Library", "Application Support", dirName), nil
	default:
		if xdg := env("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, dirName), nil
		}
		home := env("HOME")
		if home == "" {
			return "", fmt.Errorf("config: neither $XDG_CONFIG_HOME nor $HOME is defined")
		}
		return filepath.Join(home, ".config", dirName), nil
	}
}

// File returns the path of the configuration file.
func File(goos string, env func(string) string) (string, error) {
	if path := env(EnvFile); path != "" {
		return filepath.Clean(path), nil
	}
	dir, err := Dir(goos, env)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

// Exists reports whether a configuration file is present. A directory is not
// a configuration file.
func Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
