"use client";

import { useState } from "react";
import type { BranchResponse, StackDetail } from "@/lib/api";
import { BranchCard } from "@/components/branch-card";
import { FolderGit2, ChevronDown } from "lucide-react";
import Markdown from "react-markdown";

export { shortenBranchName } from "@/components/branch-card";

interface StackColumnProps {
  stack: StackDetail;
  selectedBranch: string | null;
  selectedStack: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
  onSelectStack: (stack: StackDetail) => void;
  compact?: boolean;
}

export function StackColumn({
  stack,
  selectedBranch,
  selectedStack,
  onSelectBranch,
  onSelectStack,
  compact = false,
}: StackColumnProps) {
  // Order branches root→leaf, then reverse so leaf is at top (stacking metaphor)
  const orderedBranches = orderBranches(stack.branches);
  const displayBranches = [...orderedBranches].reverse();

  return (
    <div className="flex flex-col w-64 shrink-0">
      {/* Stack header */}
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

      {/* Stacked branch cards */}
      <div className="flex flex-col bg-card rounded-t-lg">
        {displayBranches.map((branch, i) => {
          const isFirst = i === 0;
          const isLast = i === displayBranches.length - 1;

          return (
            <BranchCard
              key={branch.name}
              branch={branch}
              isSelected={selectedBranch === branch.name}
              onClick={onSelectBranch}
              compact={compact}
              className={`animate-fade-in-up border-x border-t
                ${isLast ? "border-b rounded-b-lg relative z-[1] shadow-[0_4px_6px_-2px_rgba(0,0,0,0.15)]" : ""}
                ${isFirst ? "rounded-t-lg" : ""}
              `}
              style={{ animationDelay: `${i * 50}ms` }}
            />
          );
        })}
      </div>

      {/* Status footer */}
      <StackStatusFooter
        status={stack.status}
        selected={selectedStack === stack.rootBranch}
        compact={compact}
        onClick={() => onSelectStack(stack)}
      />
    </div>
  );
}

const statusConfig: Record<string, { label: string; bg: string; selectedBg: string; text: string; shadow: string }> = {
  shippable: {
    label: "Ready to ship",
    bg: "bg-green-100 dark:bg-green-950",
    selectedBg: "bg-green-200 dark:bg-green-900",
    text: "text-green-700 dark:text-green-400",
    shadow: "shadow-[0_2px_8px_rgba(34,197,94,0.2)] dark:shadow-[0_2px_8px_rgba(34,197,94,0.15)]",
  },
  pending: {
    label: "Needs restack",
    bg: "bg-amber-100 dark:bg-amber-950",
    selectedBg: "bg-amber-200 dark:bg-amber-900",
    text: "text-amber-700 dark:text-amber-400",
    shadow: "shadow-[0_2px_8px_rgba(245,158,11,0.2)] dark:shadow-[0_2px_8px_rgba(245,158,11,0.15)]",
  },
  blocked: {
    label: "Blocked",
    bg: "bg-red-100 dark:bg-red-950",
    selectedBg: "bg-red-200 dark:bg-red-900",
    text: "text-red-700 dark:text-red-400",
    shadow: "shadow-[0_2px_8px_rgba(239,68,68,0.2)] dark:shadow-[0_2px_8px_rgba(239,68,68,0.15)]",
  },
  incomplete: {
    label: "Incomplete",
    bg: "bg-muted",
    selectedBg: "bg-muted",
    text: "text-muted-foreground",
    shadow: "shadow-[0_2px_8px_rgba(0,0,0,0.05)] dark:shadow-[0_2px_8px_rgba(0,0,0,0.15)]",
  },
};

export function StackStatusFooter({
  status,
  selected,
  onClick,
  compact = false,
}: {
  status: string;
  selected: boolean;
  onClick: () => void;
  compact?: boolean;
}) {
  const c = statusConfig[status] || statusConfig.incomplete;

  return (
    <button
      onClick={onClick}
      className={`flex items-center justify-center px-3 rounded-b-lg border-x border-b font-medium cursor-pointer transition-all duration-200 ${compact ? "-mt-1.5 pt-2.5 py-1 text-[11px]" : "-mt-2 pt-3.5 py-1.5 text-xs"} ${c.text} ${selected ? `${c.selectedBg} font-semibold` : `${c.bg} ${c.shadow} hover:brightness-95 dark:hover:brightness-110`}`}
    >
      {c.label}
    </button>
  );
}

/** Walk the branch tree root→leaf in depth-first order. */
function orderBranches(branches: BranchResponse[]): BranchResponse[] {
  const byName = new Map(branches.map((b) => [b.name, b]));
  const roots = branches.filter((b) => !b.parent || !byName.has(b.parent));
  const ordered: BranchResponse[] = [];

  function walk(name: string) {
    const branch = byName.get(name);
    if (!branch) return;
    ordered.push(branch);
    const children = branches.filter((b) => b.parent === name);
    for (const child of children) {
      walk(child.name);
    }
  }

  for (const root of roots) {
    walk(root.name);
  }

  return ordered;
}

const DESCRIPTION_COLLAPSE_LENGTH = 80;

export function StackDescription({
  text,
  compact = false,
}: {
  text: string;
  compact?: boolean;
}) {
  const canCollapse = text.length > DESCRIPTION_COLLAPSE_LENGTH;
  const [collapsed, setCollapsed] = useState(canCollapse);

  return (
    <div
      onClick={canCollapse ? () => setCollapsed(!collapsed) : undefined}
      className={`${compact ? "text-[11px] mt-0.5" : "text-xs mt-1"} text-muted-foreground flex items-start gap-1 ${canCollapse ? "cursor-pointer hover:text-foreground/70" : ""}`}
    >
      <div className={`prose prose-xs dark:prose-invert max-w-none [&>*]:text-muted-foreground [&>*:first-child]:mt-0 [&>*:last-child]:mb-0 [&_ul]:my-0.5 [&_ol]:my-0.5 [&_li]:my-0 [&_p]:my-0.5 ${compact ? "[&>*]:text-[11px]" : "[&>*]:text-xs"} ${collapsed ? "line-clamp-2" : ""}`}>
        <Markdown>{text}</Markdown>
      </div>
      {canCollapse && (
        <ChevronDown
          className={`w-3 h-3 shrink-0 mt-0.5 transition-transform duration-200 ${collapsed ? "" : "rotate-180"}`}
        />
      )}
    </div>
  );
}

/** Returns true if any branch in the list has more than one child (i.e. the stack forks). */
export function hasBranching(branches: BranchResponse[]): boolean {
  return branches.some((b) => (b.children?.length ?? 0) > 1);
}
