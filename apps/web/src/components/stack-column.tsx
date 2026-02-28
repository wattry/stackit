"use client";

import type { BranchResponse, StackDetail } from "@/lib/api";
import { PRBadge, DiffStats } from "@/components/status/status-badge";
import { AnimatedCheckmark, AnimatedX, PulsingDot } from "@/components/ui/animated-ci-icons";

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
        {stack.prCount > 0 && (
          <span className="text-xs text-muted-foreground">
            {stack.prCount} PR{stack.prCount !== 1 ? "s" : ""}
          </span>
        )}
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
      <div className="flex flex-col bg-card rounded-t-lg">
        {displayBranches.map((branch, i) => {
          const isFirst = i === 0;
          const isLast = i === displayBranches.length - 1;
          const isSelected = selectedBranch === branch.name;

          return (
            <button
              key={branch.name}
              onClick={() => onSelectBranch(branch)}
              className={`text-left px-3 py-2.5 border-x border-t transition-all duration-200 animate-fade-in-up
                ${isLast ? "border-b" : ""}
                ${isFirst ? "rounded-t-lg" : ""}
                ${isSelected ? "bg-accent ring-2 ring-ring z-10 relative shadow-[0_0_15px_var(--glow-color)]" : "hover:bg-muted/50 hover:scale-[1.02] hover:shadow-md hover:-translate-y-0.5"}
                ${branch.isCurrent && !isSelected ? "animate-breathe-glow" : ""}
              `}
              style={{ animationDelay: `${i * 50}ms` }}
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

      {/* Status footer */}
      <StackStatusFooter status={stack.status} />
    </div>
  );
}

const statusConfig: Record<string, { label: string; bg: string; text: string; shadow: string }> = {
  shippable: {
    label: "Ready to ship",
    bg: "bg-green-100 dark:bg-green-950/60",
    text: "text-green-700 dark:text-green-400",
    shadow: "shadow-[0_2px_8px_rgba(34,197,94,0.2)] dark:shadow-[0_2px_8px_rgba(34,197,94,0.15)]",
  },
  pending: {
    label: "Needs restack",
    bg: "bg-amber-100 dark:bg-amber-950/60",
    text: "text-amber-700 dark:text-amber-400",
    shadow: "shadow-[0_2px_8px_rgba(245,158,11,0.2)] dark:shadow-[0_2px_8px_rgba(245,158,11,0.15)]",
  },
  blocked: {
    label: "Blocked",
    bg: "bg-red-100 dark:bg-red-950/60",
    text: "text-red-700 dark:text-red-400",
    shadow: "shadow-[0_2px_8px_rgba(239,68,68,0.2)] dark:shadow-[0_2px_8px_rgba(239,68,68,0.15)]",
  },
  incomplete: {
    label: "Incomplete",
    bg: "bg-muted",
    text: "text-muted-foreground",
    shadow: "shadow-[0_2px_8px_rgba(0,0,0,0.05)] dark:shadow-[0_2px_8px_rgba(0,0,0,0.15)]",
  },
};

function StackStatusFooter({ status }: { status: string }) {
  const c = statusConfig[status] || statusConfig.incomplete;

  return (
    <div
      className={`flex items-center justify-center px-3 py-1.5 rounded-b-lg border-x border-b text-xs font-medium ${c.bg} ${c.text} ${c.shadow}`}
    >
      {c.label}
    </div>
  );
}

function CIStatusIcon({ status }: { status?: string }) {
  if (!status || status === "none") return null;

  switch (status) {
    case "passing":
      return <AnimatedCheckmark />;
    case "failing":
      return <AnimatedX />;
    case "pending":
      return <PulsingDot />;
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
