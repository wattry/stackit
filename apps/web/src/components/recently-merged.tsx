"use client";

import { useMemo } from "react";
import { GitMerge } from "lucide-react";
import { useRepo } from "@/components/providers/repo-provider";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import type { TrunkCommitResponse } from "@/lib/api";

function formatTimeAgo(dateStr: string): string {
  const seconds = Math.floor(
    (Date.now() - new Date(dateStr).getTime()) / 1000
  );
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function stripPRSuffix(message: string): string {
  return message.replace(/\s*\(#\d+\)\s*$/, "");
}

function prUrl(owner: string, repo: string, pr: number): string {
  return `https://github.com/${owner}/${repo}/pull/${pr}`;
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

type TimeGroup = "Today" | "Yesterday" | "This week" | "Earlier";

function timeGroup(dateStr: string): TimeGroup {
  const now = new Date();
  const date = new Date(dateStr);
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const startOfYesterday = new Date(startOfToday.getTime() - 86_400_000);
  const startOfWeek = new Date(startOfToday.getTime() - startOfToday.getDay() * 86_400_000);

  if (date >= startOfToday) return "Today";
  if (date >= startOfYesterday) return "Yesterday";
  if (date >= startOfWeek) return "This week";
  return "Earlier";
}

function groupCommits(
  commits: TrunkCommitResponse[]
): { label: TimeGroup; commits: TrunkCommitResponse[] }[] {
  const order: TimeGroup[] = ["Today", "Yesterday", "This week", "Earlier"];
  const map = new Map<TimeGroup, TrunkCommitResponse[]>();
  for (const c of commits) {
    const g = timeGroup(c.date);
    const list = map.get(g);
    if (list) {
      list.push(c);
    } else {
      map.set(g, [c]);
    }
  }
  return order.filter((l) => map.has(l)).map((l) => ({ label: l, commits: map.get(l)! }));
}

function AuthorAvatar({ name, sha }: { name: string; sha: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="flex items-center justify-center w-5 h-5 rounded-full bg-muted text-[9px] font-semibold text-muted-foreground shrink-0 select-none">
          {initials(name)}
        </span>
      </TooltipTrigger>
      <TooltipContent side="left">
        <span>{name}</span>
        <span className="ml-2 font-mono text-muted-foreground">{sha}</span>
      </TooltipContent>
    </Tooltip>
  );
}

function PRLink({
  pr,
  owner,
  repoName,
}: {
  pr: number;
  owner: string;
  repoName: string;
}) {
  return (
    <a
      href={prUrl(owner, repoName, pr)}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center text-[11px] font-mono font-medium text-muted-foreground hover:text-foreground bg-muted hover:bg-accent rounded px-1.5 py-0.5 transition-colors"
    >
      #{pr}
    </a>
  );
}

function RegularCommitItem({
  commit,
  owner,
  repoName,
}: {
  commit: TrunkCommitResponse;
  owner?: string;
  repoName?: string;
}) {
  const displayMessage = stripPRSuffix(commit.message);

  return (
    <div className="flex items-center gap-2 py-1.5 group">
      <AuthorAvatar name={commit.author} sha={commit.sha} />
      <span
        className="flex-1 min-w-0 text-xs text-foreground/80 truncate"
        title={commit.message}
      >
        {displayMessage}
      </span>
      <div className="flex items-center gap-1.5 shrink-0">
        {commit.prNumber && owner && repoName ? (
          <PRLink pr={commit.prNumber} owner={owner} repoName={repoName} />
        ) : null}
        <span className="text-[10px] text-muted-foreground/40 tabular-nums w-12 text-right">
          {formatTimeAgo(commit.date)}
        </span>
      </div>
    </div>
  );
}

function StackMergeItem({
  commit,
  owner,
  repoName,
}: {
  commit: TrunkCommitResponse;
  owner?: string;
  repoName?: string;
}) {
  const displayMessage = stripPRSuffix(commit.message);
  const stackPRs = commit.stackPRs ?? [];
  const titles = commit.stackPRTitles;
  const hasTitles = titles && Object.keys(titles).length > 0;

  return (
    <div className="py-1.5">
      {/* Main commit line */}
      <div className="flex items-center gap-2">
        <AuthorAvatar name={commit.author} sha={commit.sha} />
        <GitMerge className="h-3 w-3 shrink-0 text-purple-500 dark:text-purple-400" />
        {commit.stackScope && (
          <span className="text-[10px] font-mono text-purple-500/70 dark:text-purple-400/70 shrink-0">
            {commit.stackScope}
          </span>
        )}
        <span
          className="flex-1 min-w-0 text-xs text-foreground/80 truncate"
          title={commit.message}
        >
          {displayMessage}
        </span>
        <div className="flex items-center gap-1.5 shrink-0">
          {commit.prNumber && owner && repoName ? (
            <a
              href={prUrl(owner, repoName, commit.prNumber)}
              target="_blank"
              rel="noopener noreferrer"
              className="text-[10px] font-mono font-medium text-purple-500/60 dark:text-purple-400/50 hover:text-purple-600 dark:hover:text-purple-300 transition-colors"
            >
              #{commit.prNumber}
            </a>
          ) : (
            <span className="text-[10px] font-medium text-purple-600/70 dark:text-purple-400/60">
              {commit.stackSize} PRs
            </span>
          )}
          <span className="text-[10px] text-muted-foreground/40 tabular-nums w-12 text-right">
            {formatTimeAgo(commit.date)}
          </span>
        </div>
      </div>

      {/* Constituent PRs with titles */}
      {stackPRs.length > 0 && owner && repoName && (
        hasTitles ? (
          <div className="mt-1.5 ml-7 space-y-0.5">
            {stackPRs.map((pr) => (
              <div key={pr} className="flex items-center gap-1.5 min-w-0">
                <a
                  href={prUrl(owner, repoName, pr)}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[10px] font-mono text-purple-500/60 dark:text-purple-400/50 hover:text-purple-600 dark:hover:text-purple-300 shrink-0 transition-colors"
                >
                  #{pr}
                </a>
                {titles[pr] && (
                  <span className="text-[10px] text-muted-foreground/70 truncate">
                    {titles[pr]}
                  </span>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="flex items-center gap-1 mt-1.5 ml-7">
            {stackPRs.map((pr) => (
              <a
                key={pr}
                href={prUrl(owner, repoName, pr)}
                target="_blank"
                rel="noopener noreferrer"
                className="text-[10px] font-mono text-purple-500/60 dark:text-purple-400/50 hover:text-purple-600 dark:hover:text-purple-300 bg-purple-500/5 hover:bg-purple-500/10 rounded px-1.5 py-0.5 transition-colors"
              >
                #{pr}
              </a>
            ))}
          </div>
        )
      )}
    </div>
  );
}

function CommitItem({
  commit,
  owner,
  repoName,
}: {
  commit: TrunkCommitResponse;
  owner?: string;
  repoName?: string;
}) {
  if (commit.kind === "stack-merge") {
    return (
      <StackMergeItem commit={commit} owner={owner} repoName={repoName} />
    );
  }
  return (
    <RegularCommitItem commit={commit} owner={owner} repoName={repoName} />
  );
}

export function RecentlyMerged({ compact = false }: { compact?: boolean }) {
  const { recentlyMerged, repo } = useRepo();

  const groups = useMemo(
    () => groupCommits(recentlyMerged ?? []),
    [recentlyMerged]
  );

  if (groups.length === 0) {
    return null;
  }

  return (
    <div
      className={`${compact ? "px-4 pb-2 max-h-36 space-y-1.5" : "px-6 pb-4 max-h-64 space-y-2"} overflow-y-auto`}
    >
      {groups.map((group) => (
        <div key={group.label}>
          <div className={`${compact ? "text-[9px]" : "text-[10px]"} font-medium text-muted-foreground/50 uppercase tracking-wider mb-0.5`}>
            {group.label}
          </div>
          <div>
            {group.commits.map((commit) => (
              <CommitItem
                key={commit.sha}
                commit={commit}
                owner={repo?.owner}
                repoName={repo?.repo}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
