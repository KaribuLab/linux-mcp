package policy

import (
	"errors"
	"math"
	"os"
	"path/filepath"
)

// DefaultMaxNodes bounds the number of filesystem nodes a single Walk call
// visits, independent of how many matches the caller ultimately returns.
// 50x MaxListEntries: enough for typical project trees (node_modules/vendor
// included) without enabling a full traversal of "/".
const DefaultMaxNodes = 50_000

// WalkLimits configures Walk. The zero value applies package defaults.
type WalkLimits struct {
	MaxNodes int
}

func (l WalkLimits) withDefaults() WalkLimits {
	if l.MaxNodes <= 0 {
		l.MaxNodes = DefaultMaxNodes
	}
	return l
}

// WalkFunc is invoked for every node the walk visits that passed the path
// policy and falls within [minDepth, maxDepth]. depth is relative to root
// (root itself is depth 0).
type WalkFunc func(path string, info os.FileInfo, depth int) error

var errNodeBudgetReached = errors.New("policy: walk node budget reached")

// Walk recursively visits root and its descendants, shared by find, grep, and
// future tree-walking tools. It applies CheckPath to every visited node (a
// denied node is skipped, not fatal, and the walk continues with siblings),
// never follows symlinks while descending, and stops once limits.MaxNodes
// nodes have been visited. minDepth/maxDepth bound which depths are reported
// to fn and are applied during the walk itself (pruning descent), not as a
// post-filter; maxDepth <= 0 means unlimited.
//
// Walk returns the number of nodes visited and whether the walk was cut short
// by the node budget, independent of how many times fn matched.
func Walk(root string, limits WalkLimits, minDepth, maxDepth int, fn WalkFunc) (visited int, truncated bool, err error) {
	limits = limits.withDefaults()
	effectiveMaxDepth := maxDepth
	if effectiveMaxDepth <= 0 {
		effectiveMaxDepth = math.MaxInt
	}

	rootInfo, err := os.Lstat(root)
	if err != nil {
		return 0, false, err
	}

	var walk func(path string, info os.FileInfo, depth int) error
	walk = func(path string, info os.FileInfo, depth int) error {
		if visited >= limits.MaxNodes {
			truncated = true
			return errNodeBudgetReached
		}
		visited++

		if _, denyErr := CheckPath(path); denyErr != nil {
			// Denied (or otherwise unresolvable) node: skip it, keep walking.
			return nil
		}

		if depth >= minDepth && depth <= effectiveMaxDepth {
			if err := fn(path, info, depth); err != nil {
				return err
			}
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		if !info.IsDir() || isSymlink || depth >= effectiveMaxDepth {
			return nil
		}

		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			// Unreadable directory (permissions, race with deletion): skip
			// its subtree, keep walking siblings.
			return nil
		}
		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			childInfo, lstatErr := os.Lstat(childPath)
			if lstatErr != nil {
				continue // vanished between ReadDir and Lstat
			}
			if err := walk(childPath, childInfo, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	err = walk(root, rootInfo, 0)
	if errors.Is(err, errNodeBudgetReached) {
		err = nil
	}
	return visited, truncated, err
}
