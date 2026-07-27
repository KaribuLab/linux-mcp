#!/bin/bash
is_graphify_installed() {
    if ! command -v graphify &> /dev/null; then
        return 1
    fi
    return 0
}

if ! is_graphify_installed; then
    echo "Graphify is not installed"
    exit 1
fi

graphify hook install
cp -f scripts/githooks/post-merge.sh .git/hooks/post-merge
echo "Installed post-merge hook"
cp -f scripts/githooks/post-rewrite.sh .git/hooks/post-rewrite
echo "Installed post-rewrite hook"