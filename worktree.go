package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Worktree struct {
	Path   string
	Branch string
}

func CreateIssueWorktree(ctx context.Context, runner Runner, target WatchTarget, issueNumber int) (Worktree, error) {
	branch := fmt.Sprintf("vigilante/issue-%d", issueNumber)
	path := filepath.Join(target.Path, ".worktrees", "vigilante", fmt.Sprintf("issue-%d", issueNumber))

	if _, err := runner.Run(ctx, target.Path, "git", "worktree", "prune"); err != nil {
		return Worktree{}, err
	}
	if _, err := os.Stat(path); err == nil {
		return Worktree{}, fmt.Errorf("worktree already exists for issue #%d at %s", issueNumber, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Worktree{}, err
	}

	args := []string{"worktree", "add", "-b", branch, path, target.Branch}
	if branchExists(ctx, runner, target.Path, branch) {
		args = []string{"worktree", "add", path, branch}
	}

	if _, err := runner.Run(ctx, target.Path, "git", args...); err != nil {
		return Worktree{}, err
	}
	return Worktree{Path: path, Branch: branch}, nil
}

func RemoveWorktree(ctx context.Context, runner Runner, repoPath string, worktreePath string) error {
	_, err := runner.Run(ctx, repoPath, "git", "worktree", "remove", "--force", worktreePath)
	return err
}

func branchExists(ctx context.Context, runner Runner, repoPath string, branch string) bool {
	_, err := runner.Run(ctx, repoPath, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}
