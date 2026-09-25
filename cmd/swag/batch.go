// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"

	"github.com/bladeacer/swag/internal/i18n"
	"github.com/bladeacer/swag/pkg/sub"
)

// batchPlan is one output file of a batch run.
type batchPlan struct {
	Input  string
	Target string
	Output string
}

// batchTargets splits the target list of a batch run. A comma separates the
// names, and an empty part is skipped.
func batchTargets(format string) []string {
	var targets []string
	for _, part := range strings.Split(format, ",") {
		if name := strings.TrimSpace(part); name != "" {
			targets = append(targets, name)
		}
	}
	return targets
}

// readDir lists a directory. It is a variable so a test can force the
// failure branch of the walk.
var readDir = os.ReadDir

// planBatch lists the subtitle files under dir and the output of each. A
// name that no format claims is skipped. An empty outDir writes beside the
// input, and a set outDir keeps the relative path of the input.
func planBatch(dir, outDir string, targets []string) ([]batchPlan, error) {
	var plans []batchPlan
	root := filepath.Clean(dir)
	err := walkSubtitles(root, root, outDir, targets, &plans)
	return plans, err
}

// walkSubtitles walks one directory level and then its subdirectories.
func walkSubtitles(root, dir, outDir string, targets []string, plans *[]batchPlan) error {
	entries, err := readDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := walkSubtitles(root, path, outDir, targets, plans); err != nil {
				return err
			}
			continue
		}
		if sub.DetectFormat(path) == "" {
			continue
		}
		// The walk builds every path from the root, so the relative path is
		// the path without the root prefix.
		rel := strings.TrimPrefix(strings.TrimPrefix(path, root), string(filepath.Separator))
		base := strings.TrimSuffix(rel, filepath.Ext(rel))
		for _, target := range targets {
			*plans = append(*plans, batchPlan{Input: path, Target: target, Output: batchOutput(root, outDir, base, target)})
		}
	}
	return nil
}

// batchOutput returns the output path of one plan.
func batchOutput(root, outDir, base, target string) string {
	if outDir != "" {
		return filepath.Join(outDir, base+"."+target)
	}
	return filepath.Join(root, base+"."+target)
}

// runBatch converts every subtitle file under a directory. It reads each
// file once and writes one output per target format, and a progress bar
// counts the files.
func (c *ConvertCmd) runBatch(ictx *runContext) error {
	t := ictx.T
	targets := batchTargets(c.Format)
	if len(targets) == 0 {
		return fmt.Errorf("%s", t.S(i18n.MsgBatchFormatMissing))
	}
	plans, err := planBatch(c.Input, c.Output, targets)
	if err != nil {
		return fmt.Errorf("%s: %w", t.F(i18n.MsgBatchRead, c.Input), err)
	}
	if len(plans) == 0 {
		return fmt.Errorf("%s", t.F(i18n.MsgBatchEmpty, c.Input))
	}

	// The progress bar is decoration, so its error stays unchecked.
	bar, _ := pterm.DefaultProgressbar.WithTotal(len(plans)).WithTitle(t.F(i18n.MsgBatchTitle, len(plans))).Start()
	failed, firstErr := 0, error(nil)
	for _, plan := range plans {
		losses, err := c.convertOne(plan, t)
		switch {
		case err != nil:
			failed++
			if firstErr == nil {
				firstErr = err
			}
		case ictx.CLI.Verbose && len(losses) > 0:
			pterm.Warning.Println(t.F(i18n.MsgBatchLosses, plan.Output, len(losses)))
			for _, loss := range losses {
				pterm.Warning.Printf("  %s\n", loss)
			}
		}
		bar.Increment()
	}
	_, _ = bar.Stop()
	if failed > 0 {
		return fmt.Errorf("%s", t.F(i18n.MsgBatchFailed, failed, firstErr))
	}
	pterm.Success.Println(t.F(i18n.MsgBatchDone, len(plans)))
	return nil
}

// convertOne writes one output of a batch run.
func (c *ConvertCmd) convertOne(plan batchPlan, t *i18n.T) ([]string, error) {
	source, err := openInput(plan.Input)
	if err != nil {
		return nil, fmt.Errorf("%s", t.F(i18n.MsgInputUnreadable, err))
	}
	defer func() { _ = source.Close() }()
	data, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("%s", t.F(i18n.MsgInputUnreadable, err))
	}
	// A plan keeps the relative path of its input, so the output directory
	// of one plan can be several levels below the root.
	if err := os.MkdirAll(filepath.Dir(plan.Output), 0o755); err != nil {
		return nil, fmt.Errorf("%s: %w", t.F(i18n.MsgOutputCreate, plan.Output), err)
	}
	sink, closer, err := openOutput(plan.Output, t)
	if err != nil {
		return nil, err
	}
	defer closer()
	opts := sub.Options{Format: c.From, Target: plan.Target, Font: c.Font}
	if c.Strict {
		opts.Loss = sub.LossStrict
	}
	return sub.ConvertWith(plan.Input, bytes.NewReader(data), opts, sink)
}
