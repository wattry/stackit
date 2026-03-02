"use client";

import type { BranchResponse, StackDetail } from "@/lib/api";
import { StackTree } from "@/components/stack-tree/stack-tree";
import { StackStatusFooter, StackDescription } from "@/components/stack-column";
import { FolderGit2 } from "lucide-react";

interface StackTreeColumnProps {
  stack: StackDetail;
  selectedBranch: string | null;
  selectedStack: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
  onSelectStack: (stack: StackDetail) => void;
}

export function StackTreeColumn({
  stack,
  selectedBranch,
  selectedStack,
  onSelectBranch,
  onSelectStack,
}: StackTreeColumnProps) {
  return (
    <div className="flex flex-col min-w-64 shrink-0">
      {/* Stack header */}
      <div className="px-1 pb-2">
        {stack.hasWorktree && (
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground flex items-center gap-1" title="Worktree">
              <FolderGit2 className="w-3 h-3" />
            </span>
          </div>
        )}
        {stack.title && (
          <p className="text-xs font-medium text-muted-foreground mt-1 truncate" title={stack.title}>
            {stack.title}
          </p>
        )}
        {stack.description && (
          <StackDescription text={stack.description} />
        )}
      </div>

      {/* Tree visualization */}
      <div className="bg-card rounded-t-lg border border-b-0 p-2">
        <StackTree
          branches={stack.branches}
          selectedBranch={selectedBranch}
          onSelectBranch={onSelectBranch}
        />
      </div>

      {/* Status footer */}
      <StackStatusFooter
        status={stack.status}
        selected={selectedStack === stack.rootBranch}
        onClick={() => onSelectStack(stack)}
      />
    </div>
  );
}
