// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/config"
	"github.com/bladeacer/swag/internal/i18n"
)

// configFile resolves the configuration file path. It is a variable so a
// test can force the failure branch.
var configFile = config.File

// ConfigCmd reports where the tool looks for its configuration file, and
// whether the file is present.
type ConfigCmd struct{}

// Run resolves the location and prints it with the platform of the build.
func (c *ConfigCmd) Run(ictx *runContext) error {
	t := ictx.T
	path, err := configFile(runtime.GOOS, os.Getenv)
	if err != nil {
		return fmt.Errorf("%s", t.F(i18n.MsgConfigError, err))
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
