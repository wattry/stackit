"use client";

import type { BranchResponse, StackDetail } from "@/lib/api";
import { PRBadge, DiffStats, StackStatusBadge } from "@/components/status/status-badge";

interface StackColumnProps {
  stack: StackDetail;
  selectedBranch: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
}

export function StackColumn({
  stack,
  selectedBranch,
  onSelectBranch,
}: StackColumnProps) {
  // Order branches root→leaf, then reverse so leaf is at top (stacking metaphor)
  const orderedBranches = orderBranches(stack.branches);
  const displayBranches = [...orderedBranches].reverse();

  return (
    <div className="flex flex-col w-64 shrink-0">
      {/* Stack header */}
      <div className="px-1 pb-2">
        <div className="flex items-center gap-2">
          <StackStatusBadge status={stack.status} />
          {stack.prCount > 0 && (
            <span className="text-xs text-muted-foreground">
              {stack.prCount} PR{stack.prCount !== 1 ? "s" : ""}
            </span>
          )}
        </div>
        {stack.title && (
          <p className="text-xs text-muted-foreground mt-1 truncate" title={stack.title}>
            {stack.title}
          </p>
        )}
        {stack.description && (
          <p className="text-xs text-muted-foreground/60 mt-1 line-clamp-2" title={stack.description}>
            {stack.description}
          </p>
        )}
      </div>

      {/* Stacked branch cards */}
      <div className="flex flex-col bg-white rounded-lg">
        {displayBranches.map((branch, i) => {
          const isFirst = i === 0;
          const isLast = i === displayBranches.length - 1;
          const isSelected = selectedBranch === branch.name;

          return (
            <button
              key={branch.name}
              onClick={() => onSelectBranch(branch)}
              className={`text-left px-3 py-2.5 border-x border-t transition-colors
                ${isLast ? "border-b" : ""}
                ${isFirst ? "rounded-t-lg" : ""}
                ${isLast ? "rounded-b-lg" : ""}
                ${isSelected ? "bg-accent ring-2 ring-ring z-10 relative" : "hover:bg-muted/50"}
              `}
            >
              <div className="flex items-center gap-2 min-w-0">
                {branch.isCurrent && (
                  <span className="w-2 h-2 rounded-full bg-amber-500 shrink-0" />
                )}
                <span
                  className="text-sm font-medium truncate"
                  title={branch.name}
                >
                  {branch.commits?.[0]?.message || shortenBranchName(branch.name)}
                </span>
                {branch.needsRestack && (
                  <span className="text-amber-500 shrink-0" title="Needs restack">
                    &#x21BB;
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2 mt-1">
                {branch.pr ? (
                  <PRBadge pr={branch.pr} />
                ) : (
                  <span className="text-xs text-muted-foreground">no PR</span>
                )}
                <CIStatusIcon status={branch.ci?.status} />
                <DiffStats added={branch.linesAdded} deleted={branch.linesDeleted} />
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function CIStatusIcon({ status }: { status?: string }) {
  if (!status || status === "none") return null;

  switch (status) {
    case "passing":
      return (
        <span className="text-green-600 text-xs" title="CI passing">
          &#x2713;
        </span>
      );
    case "failing":
      return (
        <span className="text-red-600 text-xs" title="CI failing">
          &#x2717;
        </span>
      );
    case "pending":
      return (
        <span className="text-amber-500 text-xs" title="CI pending">
          &#x23F3;
        </span>
      );
    default:
      return null;
  }
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

/** Strip common prefixes like "user/timestamp/" to show a shorter name. */
function shortenBranchName(name: string): string {
  // Match patterns like "user/timestamp/description" and return the description part
  const match = name.match(/^[^/]+\/\d{14}\/(.+)$/);
  return match ? match[1] : name;
}
