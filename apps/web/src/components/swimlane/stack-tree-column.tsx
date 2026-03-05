"use client";

import type { BranchResponse, StackDetail } from "@/lib/api";
import { SegmentTree } from "@/components/stack-tree/segment-tree";
import { StackStatusFooter, StackDescription } from "@/components/swimlane/stack-column";
import { FolderGit2 } from "lucide-react";

interface StackTreeColumnProps {
  stack: StackDetail;
  selectedBranch: string | null;
  selectedStack: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
  onSelectStack: (stack: StackDetail) => void;
  compact?: boolean;
  showHeader?: boolean;
  showFooter?: boolean;
}

export function StackTreeColumn({
  stack,
  selectedBranch,
  selectedStack,
  onSelectBranch,
  onSelectStack,
  compact = false,
  showHeader = true,
  showFooter = true,
}: StackTreeColumnProps) {
  return (
    <div className="flex flex-col min-w-64 shrink-0">
      {/* Stack header */}
      {showHeader && (
        <div className={`px-1 ${compact ? "pb-1" : "pb-2"}`}>
          {stack.hasWorktree && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground flex items-center gap-1" title="Worktree">
                <FolderGit2 className="w-3 h-3" />
              </span>
            </div>
          )}
          {stack.title && (
            <p className={`font-medium text-muted-foreground truncate ${compact ? "text-[11px] mt-0.5" : "text-xs mt-1"}`} title={stack.title}>
              {stack.title}
            </p>
          )}
          {stack.description && (
            <StackDescription text={stack.description} compact={compact} />
          )}
        </div>
      )}

      {/* Tree visualization: linear card stacks with connectors at branch points */}
      <SegmentTree
        branches={stack.branches}
        selectedBranch={selectedBranch}
        onSelectBranch={onSelectBranch}
        compact={compact}
        footer={showFooter ? (
          <StackStatusFooter
            status={stack.status}
            selected={selectedStack === stack.rootBranch}
            compact={compact}
            onClick={() => onSelectStack(stack)}
          />
        ) : undefined}
      />
    </div>
  );
}
