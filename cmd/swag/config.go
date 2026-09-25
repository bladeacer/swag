// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/config"
	"github.com/bladeacer/swag/internal/i18n"
)

// configFile resolves the configuration file path. It is a variable so a
// test can force the failure branch.
var configFile = config.File

// settingsFromFile loads the configuration file. A missing file returns the
// defaults and no error, so the tool works with no configuration.
func settingsFromFile() (config.Settings, error) {
	path, err := configFile(runtime.GOOS, os.Getenv)
	if err != nil {
		return config.Settings{}, err
	}
	return config.Load(path)
}

// configResolver returns a kong resolver over the settings. The command
// line wins over the environment, the environment wins over the file, and
// the file wins over the built-in default. Kong consults a resolver only for
// a flag that the command line left unset, and the environment check keeps
// an exported variable ahead of the file.
func configResolver(settings config.Settings) kong.Resolver {
	return kong.ResolverFunc(func(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (any, error) {
		if envSet(flag.Envs) {
			return nil, nil
		}
		value, ok := settings.Flag(flag.Name)
		if !ok {
			return nil, nil
		}
		return value, nil
	})
}

// envSet reports whether one of the environment variables of a flag holds a
// value.
func envSet(names []string) bool {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); ok {
			return true
		}
	}
	return false
}

// ConfigCmd reports where the tool looks for its configuration file, whether
// the file is present, and the platform of the build. The --init flag writes
// the default file instead.
type ConfigCmd struct {
	Init bool `help:"Write the default configuration file and report where it lands." short:"i"`
}

// Run resolves the location. With --init it writes the commented default
// file, and otherwise it reports the location and the platform.
func (c *ConfigCmd) Run(ictx *runContext) error {
	t := ictx.T
	path, err := configFile(runtime.GOOS, os.Getenv)
	if err != nil {
		return fmt.Errorf("%s", t.F(i18n.MsgConfigError, err))
	}
	if c.Init {
		if config.Exists(path) {
			return fmt.Errorf("%s", t.F(i18n.MsgConfigExists, path))
		}
		if err := config.Write(path, config.DefaultFile); err != nil {
			return fmt.Errorf("%s", t.F(i18n.MsgConfigWrite, err))
		}
		pterm.Success.Println(t.F(i18n.MsgConfigInit, path))
		return nil
	}
	pterm.Info.Println(t.F(i18n.MsgConfigFile, path))
	if config.Exists(path) {
		pterm.Success.Println(t.S(i18n.MsgConfigPresent))
	} else {
		pterm.Info.Println(t.S(i18n.MsgConfigAbsent))
	}
	pterm.Info.Println(t.F(i18n.MsgConfigPlatform, runtime.GOOS, runtime.GOARCH))
	return nil
}
