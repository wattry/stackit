package git

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-git/go-git/v6/plumbing"
)

func (r *runner) GetCurrentBranch() (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not on a branch")
	}
	return head.Name().Short(), nil
}

func (r *runner) GetAllBranchNames() ([]string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}
	branches, err := r.allBranchHashes(repo)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(branches))
	for name := range branches {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func (r *runner) CheckoutBranch(ctx context.Context, branchName string) error {
	_, err := r.RunGitCommandWithContext(ctx, "checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}
	return nil
}

func (r *runner) CheckoutBranchForce(ctx context.Context, branchName string) error {
	_, err := r.RunGitCommandWithContext(ctx, "checkout", "-f", branchName)
	return err
}

func (r *runner) CreateAndCheckoutBranch(ctx context.Context, branchName string) error {
	_, err := r.RunGitCommandWithContext(ctx, "checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create and checkout branch %s: %w", branchName, err)
	}
	return nil
}

func (r *runner) DeleteBranch(ctx context.Context, branchName string) error {
	_, err := r.RunGitCommandWithContext(ctx, "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}
	r.revisionCache.Delete(branchName)
	return nil
}

func (r *runner) RenameBranch(ctx context.Context, oldName, newName string) error {
	_, err := r.RunGitCommandWithContext(ctx, "branch", "-m", oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename branch %s to %s: %w", oldName, newName, err)
	}
	r.revisionCache.Delete(oldName)
	r.revisionCache.Delete(newName)
	return nil
}

func (r *runner) CreateBranch(ctx context.Context, branchName, startPoint string) error {
	_, err := r.RunGitCommandWithContext(ctx, "branch", branchName, startPoint)
	if err != nil {
		return fmt.Errorf("failed to create branch %s from %s: %w", branchName, startPoint, err)
	}
	return nil
}

func (r *runner) CreateBranchForce(ctx context.Context, branchName, revision string) error {
	_, err := r.RunGitCommandWithContext(ctx, "branch", "-f", branchName, revision)
	if err == nil {
		r.revisionCache.Delete(branchName)
	}
	return err
}

func (r *runner) CheckoutDetached(ctx context.Context, revision string) error {
	_, err := r.RunGitCommandWithContext(ctx, "checkout", "--detach", revision)
	if err != nil {
		return fmt.Errorf("failed to checkout %s in detached state: %w", revision, err)
	}
	return nil
}

func (r *runner) UpdateBranchRef(ctx context.Context, branchName, revision string) error {
	_, err := r.RunGitCommandWithContext(ctx, "update-ref", "refs/heads/"+branchName, revision)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}
	r.revisionCache.Delete(branchName)
	return nil
}

func (r *runner) GetCurrentBranchOrSHA(ctx context.Context) (string, error) {
	branch, err := r.GetCurrentBranch()
	if err == nil {
		return branch, nil
	}
	return r.GetCurrentRevision(ctx)
}

func (r *runner) GetMergedBranches(_ context.Context, target string) (map[string]bool, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}

	targetHash, err := r.resolveRefHash(repo, target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target %s: %w", target, err)
	}

	branches, err := r.allBranchHashes(repo)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]bool)
	for branchName, branchHash := range branches {
		if branchHash == targetHash {
			merged[branchName] = true
			continue
		}
		isMerged, err := r.isAncestorGoGit(repo, branchHash, targetHash)
		if err != nil {
			return nil, fmt.Errorf("failed to check if %s is merged into %s: %w", branchName, target, err)
		}
		if isMerged {
			merged[branchName] = true
		}
	}
	return merged, nil
}

func (r *runner) allBranchHashes(repo *Repository) (map[string]plumbing.Hash, error) {
	iter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}
	defer iter.Close()

	branches := make(map[string]plumbing.Hash)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branches[ref.Name().Short()] = ref.Hash()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}
	return branches, nil
}
