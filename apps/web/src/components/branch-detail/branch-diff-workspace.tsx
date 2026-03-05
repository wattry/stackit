"use client";

import type { BranchResponse, StackDetail } from "@/lib/api";
import { DiffStats } from "@/components/status/status-badge";
import { BranchDiff } from "./branch-diff";

interface BranchDiffWorkspaceProps {
  branch: BranchResponse;
  stack: StackDetail;
  onExit: () => void;
}

export function BranchDiffWorkspace({
  branch,
  stack,
  onExit,
}: BranchDiffWorkspaceProps) {
  const title = branch.pr?.title || branch.name;

  return (
    <div className="h-full flex flex-col p-6 gap-4">
      <div className="rounded-xl border bg-card p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="text-xs uppercase tracking-wider text-muted-foreground">
              Single Stack View
            </p>
            <h2 className="text-lg font-semibold truncate" title={title}>
              {title}
            </h2>
            <p
              className="text-xs font-mono text-muted-foreground truncate mt-1"
              title={branch.name}
            >
              {branch.name}
            </p>
            <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground mt-2">
              <span>{stack.title || stack.rootBranch}</span>
              <span>{stack.branches.length} branches</span>
              <span>
                {branch.commitCount} commit{branch.commitCount !== 1 && "s"}
              </span>
              <DiffStats
                added={branch.linesAdded}
                deleted={branch.linesDeleted}
              />
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {branch.pr?.url && (
              <a
                href={branch.pr.url}
                target="_blank"
                rel="noopener noreferrer"
                className="rounded-md border px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
              >
                Open PR
              </a>
            )}
            <button
              onClick={onExit}
              className="rounded-md border px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            >
              Back to all stacks
            </button>
          </div>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto rounded-xl border bg-card p-4">
        <BranchDiff
          key={`${branch.name}:${branch.revision}`}
          branchName={branch.name}
          revision={branch.revision}
        />
      </div>
    </div>
  );
}
