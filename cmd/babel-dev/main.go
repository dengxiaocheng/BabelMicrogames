package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"babel-runtime/internal/corehost"
	"babel-runtime/internal/devtools/quality"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var exitCode int
	switch os.Args[1] {
	case "check-docs-consistency":
		exitCode = runCheckDocsConsistency(repoRoot)
	case "check-docs-sync-guard":
		exitCode = runCheckDocsSyncGuard(repoRoot, os.Args[2:])
	case "check-requirement-assets":
		exitCode = runCheckRequirementAssets(repoRoot)
	case "install-ops-binaries":
		exitCode = runInstallOpsBinaries(repoRoot)
	case "scene-host-fixture":
		exitCode = runSceneHostFixture(os.Args[2:])
	case "verify-scene-host-library":
		exitCode = runVerifySceneHostLibrary(os.Args[2:])
	case "render-guard-report":
		exitCode = runRenderGuardReport(os.Args[2:])
	case "install-hooks":
		exitCode = runInstallHooks(repoRoot)
	default:
		usage()
		exitCode = 2
	}
	os.Exit(exitCode)
}

func runCheckDocsConsistency(repoRoot string) int {
	errors, err := quality.ValidateDocsConsistency(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(errors) > 0 {
		for _, message := range errors {
			fmt.Println(message)
		}
		return 1
	}
	fmt.Println("docs consistency OK")
	return 0
}

func runCheckDocsSyncGuard(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("check-docs-sync-guard", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	staged := fs.Bool("staged", false, "只检查 staged 变更，适合 pre-commit hook。")
	against := fs.String("against", "HEAD", "未指定 --staged 时，使用哪个 git ref 作为 diff 基线。默认 HEAD。")
	base := fs.String("base", "", "显式指定 diff base commit/ref。常用于 CI。")
	head := fs.String("head", "HEAD", "与 --base 配对使用的 head commit/ref。默认 HEAD。")
	mergeBaseWith := fs.String("merge-base-with", "", "先与给定 ref 计算 merge-base，再以该 merge-base 作为 diff base。适合 PR CI。")
	var paths stringSliceFlag
	fs.Var(&paths, "path", "显式提供变更路径，可重复传入，适合测试或手动检查。")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	manifest, err := quality.LoadManifest(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var changedPaths []string
	if len(paths) > 0 {
		changedPaths = quality.NormalizeRepoPaths(repoRoot, paths)
	} else {
		changedPaths, err = quality.ChangedPathsFromGit(repoRoot, quality.GitDiffOptions{
			Staged:        *staged,
			Against:       *against,
			Base:          *base,
			Head:          *head,
			MergeBaseWith: *mergeBaseWith,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	if len(changedPaths) == 0 {
		fmt.Println("docs sync guard OK (no changes)")
		return 0
	}
	errors := quality.ValidateSyncGuard(repoRoot, changedPaths, manifest)
	if len(errors) > 0 {
		for _, message := range errors {
			fmt.Println(message)
		}
		return 1
	}
	fmt.Println("docs sync guard OK")
	return 0
}

func runCheckRequirementAssets(repoRoot string) int {
	errors := quality.ValidateRequirementAssets(repoRoot)
	if len(errors) > 0 {
		for _, message := range errors {
			fmt.Println(message)
		}
		return 1
	}
	fmt.Println("requirement assets OK")
	return 0
}

func runRenderGuardReport(args []string) int {
	fs := flag.NewFlagSet("render-guard-report", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	title := fs.String("title", "docs-sync-guard report", "报告标题。")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: babel-dev render-guard-report [--title TITLE] <status_file>")
		return 2
	}

	entries, err := quality.LoadGuardEntries(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(quality.RenderGuardReport(entries, *title))
	return 0
}

func runSceneHostFixture(args []string) int {
	fs := flag.NewFlagSet("scene-host-fixture", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	mode := fs.String("mode", "", "fixture mode: solo_step or room_step")
	kind := fs.String("kind", "request", "fixture kind: request or response")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *mode == "" {
		fmt.Fprintln(os.Stderr, "missing --mode")
		return 2
	}

	request, response, err := corehost.FixturePair(*mode)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var payload []byte
	switch *kind {
	case "request":
		payload, err = corehost.MarshalFixture(request)
	case "response":
		payload, err = corehost.MarshalFixture(response)
	default:
		fmt.Fprintln(os.Stderr, "invalid --kind, want request or response")
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(string(payload))
	return 0
}

func runVerifySceneHostLibrary(args []string) int {
	fs := flag.NewFlagSet("verify-scene-host-library", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	library := fs.String("library", "", "shared library path")
	mode := fs.String("mode", "all", "verification mode: solo_step, room_step, or all")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *library == "" {
		fmt.Fprintln(os.Stderr, "missing --library")
		return 2
	}

	modes := []string{*mode}
	if *mode == "all" {
		modes = []string{"solo_step", "room_step"}
	}
	if err := corehost.VerifyFixtureLibrary(context.Background(), *library, modes...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, operation := range modes {
		fmt.Printf("%s verification OK\n", operation)
	}
	return 0
}

func runInstallHooks(repoRoot string) int {
	cmd := exec.Command("git", "-C", repoRoot, "config", "core.hooksPath", ".githooks")
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Installed local git hooks at %s/.githooks\n", repoRoot)
	return 0
}

func runInstallOpsBinaries(repoRoot string) int {
	outputDir := repoRoot + "/.codex-runtime/bin"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	target := outputDir + "/babel-issue-bridge"
	cmd := exec.Command("go", "build", "-o", target, "./cmd/babel-issue-bridge")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return 1
	}
	fmt.Printf("Installed ops binary: %s\n", target)
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: babel-dev <check-docs-consistency|check-docs-sync-guard|check-requirement-assets|install-ops-binaries|scene-host-fixture|verify-scene-host-library|render-guard-report|install-hooks> [args]")
}

type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
