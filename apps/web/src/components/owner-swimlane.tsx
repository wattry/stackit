"use client";

import { useState } from "react";
import type { BranchResponse, StackDetail } from "@/lib/api";
import { StackColumn } from "@/components/stack-column";
import { SwimlaneLabel, swimlaneColor } from "@/components/swimlane-label";

const COLLAPSED_LIMIT = 3;

interface OwnerSwimlaneProps {
  label: string;
  lastActive?: Date;
  stacks: StackDetail[];
  selectedBranch: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
}

export function OwnerSwimlane({
  label,
  lastActive,
  stacks,
  selectedBranch,
  onSelectBranch,
}: OwnerSwimlaneProps) {
  const [expanded, setExpanded] = useState(false);
  const color = swimlaneColor(label);

  const canCollapse = stacks.length > COLLAPSED_LIMIT;
  const visibleStacks = canCollapse && !expanded
    ? stacks.slice(0, COLLAPSED_LIMIT)
    : stacks;
  const hiddenCount = stacks.length - COLLAPSED_LIMIT;

  return (
    <div className="flex flex-col shrink-0">
      <div
        className="flex gap-4 items-end px-3 pt-3"
        style={{ backgroundColor: color }}
      >
        {visibleStacks.map((stack) => (
          <StackColumn
            key={stack.rootBranch}
            stack={stack}
            selectedBranch={selectedBranch}
            onSelectBranch={onSelectBranch}
          />
        ))}
        {canCollapse && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex items-center justify-center w-8 shrink-0 self-stretch rounded border border-dashed border-muted-foreground/30 text-xs text-muted-foreground hover:border-muted-foreground/50 hover:text-foreground transition-colors"
          >
            <span className="-rotate-90 whitespace-nowrap">
              {expanded ? "Show less" : `+${hiddenCount} more`}
            </span>
          </button>
        )}
      </div>
      <SwimlaneLabel label={label} lastActive={lastActive} color={color} />
    </div>
  );
}

export function getLastActiveDate(stacks: StackDetail[]): Date | undefined {
  let latest = 0;
  for (const stack of stacks) {
    for (const branch of stack.branches) {
      const t = new Date(branch.commitDate).getTime();
      if (t > latest) latest = t;
    }
  }
  return latest ? new Date(latest) : undefined;
}
