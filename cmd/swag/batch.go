// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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

// defaultJobs returns the worker count of a batch run that names none. Two
// cores stay free, so a long conversion does not make the machine crawl.
func defaultJobs(cores int) int {
	if cores <= 2 {
		return 1
	}
	return cores - 2
}

// batchResult carries the outcome of one plan out of a worker. The index
// keeps the report in plan order, however the workers interleave.
type batchResult struct {
	index  int
	losses []string
	err    error
}

// convertAll converts the plans with the given number of workers. One
// outcome comes back for each plan, in plan order, so the same input always
// reports the same failure. tick runs once per finished plan, which keeps a
// progress bar on a single goroutine.
func (c *ConvertCmd) convertAll(plans []batchPlan, workers int, t *i18n.T, tick func()) []batchResult {
	work := make(chan int)
	done := make(chan batchResult, len(plans))
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range work {
				losses, err := c.convertOne(plans[index], t)
				done <- batchResult{index: index, losses: losses, err: err}
			}
		}()
	}
	go func() {
		for index := range plans {
			work <- index
		}
		close(work)
	}()
	go func() {
		group.Wait()
		close(done)
	}()

	results := make([]batchResult, len(plans))
	for result := range done {
		results[result.index] = result
		tick()
	}
	return results
}

// runBatch converts every subtitle file under a directory. It reads each
// file once and writes one output per target format, and a progress bar
// counts the files.
func (c *ConvertCmd) runBatch(ictx *runContext) error {
	t := ictx.T
	targets := batchTargets(c.Format)
	// A batch run without -f falls back to the preferred formats of the
	// configuration file.
	if len(targets) == 0 {
		targets = ictx.Settings.Preferred
	}
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

	// A negative count names no useful number of workers, and zero uses
	// every core. The workers never outnumber the plans.
	if c.Jobs < 0 {
		return fmt.Errorf("%s", t.F(i18n.MsgBatchJobs, c.Jobs))
	}
	workers := c.Jobs
	if workers == 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(plans) {
		workers = len(plans)
	}

	// The progress bar is decoration, so its error stays unchecked.
	bar, _ := pterm.DefaultProgressbar.WithTotal(len(plans)).WithTitle(t.F(i18n.MsgBatchTitle, len(plans))).Start()
	results := c.convertAll(plans, workers, t, func() { bar.Increment() })
	_, _ = bar.Stop()

	failed, firstErr := 0, error(nil)
	for _, result := range results {
		switch {
		case result.err != nil:
			failed++
			if firstErr == nil {
				firstErr = result.err
			}
		case ictx.CLI.Verbose && len(result.losses) > 0:
			pterm.Warning.Println(t.F(i18n.MsgBatchLosses, plans[result.index].Output, len(result.losses)))
			for _, loss := range result.losses {
				pterm.Warning.Printf("  %s\n", loss)
			}
		}
	}
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
	opts := sub.Options{Format: c.From, Target: plan.Target, Font: c.Font, StrictCompat: c.StrictCompat}
	if c.Strict {
		opts.Loss = sub.LossStrict
	}
	return sub.ConvertWith(plan.Input, bytes.NewReader(data), opts, sink)
}
