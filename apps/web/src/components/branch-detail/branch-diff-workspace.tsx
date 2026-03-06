"use client";

import dynamic from "next/dynamic";
import type { BranchResponse } from "@/lib/api";

const BranchDiff = dynamic(
  () => import("./branch-diff").then((m) => m.BranchDiff),
  {
    ssr: false,
    loading: () => (
      <p className="text-sm text-muted-foreground">Loading diff viewer...</p>
    ),
  }
);

interface BranchDiffWorkspaceProps {
  branch: BranchResponse;
  onExit: () => void;
}

export function BranchDiffWorkspace({
  branch,
  onExit,
}: BranchDiffWorkspaceProps) {
  return (
    <div className="flex h-full min-h-0 flex-col p-4">
      <div className="min-h-0 flex-1 overflow-hidden rounded-xl border bg-card p-4">
        <BranchDiff
          key={`${branch.name}:${branch.revision}`}
          branchName={branch.name}
          revision={branch.revision}
          commits={branch.commits}
          onExit={onExit}
        />
      </div>
    </div>
  );
}
