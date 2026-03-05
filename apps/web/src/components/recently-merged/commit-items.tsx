"use client";

import { GitMerge } from "lucide-react";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { prUrl } from "@/lib/github";
import { formatTimeAgo } from "@/lib/time";
import type { TrunkCommitResponse } from "@/lib/api";
import { stripPRSuffix, initials } from "./utils";

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

export function RegularCommitItem({
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

export function StackMergeItem({
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

export function CommitItem({
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
