"use client";

import type { TrunkCommitResponse } from "@/lib/api";

interface RecentlyMergedProps {
  commits: TrunkCommitResponse[];
  owner?: string;
  repo?: string;
}

export function RecentlyMerged({ commits, owner, repo }: RecentlyMergedProps) {
  if (commits.length === 0) return null;

  return (
    <div className="px-6 pb-6 space-y-1">
      {commits.map((commit) =>
        commit.stackSize && commit.stackSize > 0 ? (
          <StackMergeEntry
            key={commit.sha}
            commit={commit}
            owner={owner}
            repo={repo}
          />
        ) : (
          <RegularCommitEntry
            key={commit.sha}
            commit={commit}
            owner={owner}
            repo={repo}
          />
        )
      )}
    </div>
  );
}

function StackMergeEntry({
  commit,
  owner,
  repo,
}: {
  commit: TrunkCommitResponse;
  owner?: string;
  repo?: string;
}) {
  const prCount = commit.stackPRs?.length ?? commit.stackSize ?? 0;

  return (
    <div className="rounded-lg border border-muted p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {commit.stackScope && (
            <span className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded">
              {commit.stackScope}
            </span>
          )}
          <span className="text-sm text-foreground">
            {prCount} PR{prCount !== 1 ? "s" : ""} merged
          </span>
        </div>
        <span className="text-xs text-muted-foreground">
          {formatTimeAgo(commit.date)}
        </span>
      </div>
      {commit.stackPRs && commit.stackPRs.length > 0 && (
        <div className="flex items-center gap-1.5 mt-1.5">
          {commit.stackPRs.map((prNum) => (
            <PRLink
              key={prNum}
              prNumber={prNum}
              owner={owner}
              repo={repo}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function RegularCommitEntry({
  commit,
  owner,
  repo,
}: {
  commit: TrunkCommitResponse;
  owner?: string;
  repo?: string;
}) {
  return (
    <div className="flex items-center gap-2 py-1 px-1 text-sm text-muted-foreground">
      <span className="font-mono text-xs">{commit.sha}</span>
      <span className="truncate flex-1">
        {commit.prNumber ? (
          <>
            {stripPRSuffix(commit.message)}{" "}
            <PRLink prNumber={commit.prNumber} owner={owner} repo={repo} />
          </>
        ) : (
          commit.message
        )}
      </span>
      <span className="text-xs shrink-0">{formatTimeAgo(commit.date)}</span>
    </div>
  );
}

function PRLink({
  prNumber,
  owner,
  repo,
}: {
  prNumber: number;
  owner?: string;
  repo?: string;
}) {
  if (owner && repo) {
    return (
      <a
        href={`https://github.com/${owner}/${repo}/pull/${prNumber}`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-xs font-mono text-blue-500 hover:underline"
      >
        #{prNumber}
      </a>
    );
  }
  return <span className="text-xs font-mono text-blue-500">#{prNumber}</span>;
}

function stripPRSuffix(message: string): string {
  return message.replace(/\s*\(#\d+\)\s*$/, "");
}

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 5) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}
