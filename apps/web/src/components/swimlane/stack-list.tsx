"use client";

import type { StackSummary } from "@/lib/api";
import { StackStatusBadge } from "@/components/status/status-badge";

interface StackListProps {
  stacks: StackSummary[];
  selectedRoot: string | null;
  onSelect: (rootBranch: string) => void;
}

export function StackList({ stacks, selectedRoot, onSelect }: StackListProps) {
  if (stacks.length === 0) {
    return (
      <div className="p-4 text-sm text-muted-foreground">No stacks found</div>
    );
  }

  return (
    <div className="space-y-1 p-2">
      {stacks.map((stack) => (
        <button
          key={stack.rootBranch}
          onClick={() => onSelect(stack.rootBranch)}
          className={`w-full text-left rounded-lg p-3 transition-colors ${
            selectedRoot === stack.rootBranch
              ? "bg-accent text-accent-foreground"
              : "hover:bg-muted"
          }`}
        >
          <div className="flex items-center justify-between mb-1">
            <span className="font-mono text-sm font-medium truncate">
              {stack.rootBranch}
            </span>
            {stack.isCurrent && (
              <span className="w-2 h-2 rounded-full bg-amber-500 shrink-0 ml-2" />
            )}
          </div>
          <div className="flex items-center gap-2">
            <StackStatusBadge status={stack.status} />
            <span className="text-xs text-muted-foreground">
              {stack.branchCount} branch{stack.branchCount !== 1 && "es"}
              {stack.prCount > 0 && ` \u00B7 ${stack.prCount} PR${stack.prCount !== 1 ? "s" : ""}`}
            </span>
          </div>
          {stack.scope && (
            <div className="text-xs text-muted-foreground mt-1">
              [{stack.scope}]
            </div>
          )}
        </button>
      ))}
    </div>
  );
}
