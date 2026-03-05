"use client";

import type { BranchResponse } from "@/lib/api";
import { BranchDiff } from "./branch-diff";

interface BranchDiffWorkspaceProps {
  branch: BranchResponse;
  onExit: () => void;
}

export function BranchDiffWorkspace({
  branch,
  onExit,
}: BranchDiffWorkspaceProps) {
  return (
    <div className="h-full flex flex-col p-4 gap-3">
      <div className="flex justify-end">
        <button
          onClick={onExit}
          className="rounded-md border px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
        >
          Back to all stacks
        </button>
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
