"use client";

import type { BranchResponse, StackDetail } from "@/lib/api";
import { SegmentTree } from "@/components/stack-tree/segment-tree";
import { StackStatusFooter, StackHeader } from "@/components/swimlane/stack-column";

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
      {showHeader && (
        <StackHeader stack={stack} compact={compact} />
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
