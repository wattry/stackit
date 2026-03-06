"use client";

import type { CommitResponse } from "@/lib/api";
import { useRepo } from "@/components/providers/repo-provider";
import { commitUrl } from "@/lib/github";

export function CommitList({ commits }: { commits?: CommitResponse[] }) {
  const { repo } = useRepo();

  if (!commits || commits.length === 0) return null;

  return (
    <div className="space-y-1">
      <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
        Commits
      </h4>
      <div className="space-y-0.5">
        {commits.map((commit, i) => (
          <div
            key={commit.sha}
            className="flex items-baseline gap-2 text-sm"
          >
            {repo ? (
              <a
                href={commitUrl(repo.owner, repo.repo, commit.sha)}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-muted-foreground font-mono shrink-0 hover:text-foreground hover:underline"
              >
                {commit.sha}
              </a>
            ) : (
              <code className="text-xs text-muted-foreground font-mono shrink-0">
                {commit.sha}
              </code>
            )}
            <span className="truncate">{commit.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
