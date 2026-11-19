// Package ciworkflows checks the repository's GitHub Actions workflows.
//
// The repository is public: any GitHub user can comment on its issues and
// pull requests. A workflow started by a comment must check who wrote it,
// and every third-party action must be pinned to a full commit SHA, so that
// code we have not reviewed never runs with the repository's secrets.
package ciworkflows

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	commentTrigger = regexp.MustCompile(`(?m)^\s*(issue_comment|pull_request_review_comment|pull_request_target|discussion_comment)\s*:`)
	usesLine       = regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*["']?([^"'\s#]+)`)
	pinnedSHA      = regexp.MustCompile(`@[0-9a-f]{40}$`)
)

func workflowFiles(t *testing.T) map[string]string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", ".github", "workflows")
	files := map[string]string{}
	for _, pattern := range []string{"*.yml", "*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files[filepath.Base(path)] = string(data)
		}
	}
	return files
}

func TestCommentTriggeredWorkflowsCheckTheAuthor(t *testing.T) {
	for name, text := range workflowFiles(t) {
		if commentTrigger.MatchString(text) && !strings.Contains(text, "author_association") {
			t.Errorf("%s runs on comments from anyone: gate the job on github.event.comment.author_association", name)
		}
	}
}

func TestActionsArePinnedToCommitSHA(t *testing.T) {
	for name, text := range workflowFiles(t) {
		for _, match := range usesLine.FindAllStringSubmatch(text, -1) {
			ref := match[1]
			if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") {
				continue
			}
			if !pinnedSHA.MatchString(ref) {
				t.Errorf("%s uses %s: pin third-party actions to a full commit SHA", name, ref)
			}
		}
	}
}
