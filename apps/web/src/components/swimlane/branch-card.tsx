"use client";

import type { BranchResponse } from "@/lib/api";
import { PRBadge, DiffStats } from "@/components/status/status-badge";
import { CIStatusWithTooltip } from "@/components/status/ci-status";
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip";
import { Lock } from "lucide-react";
import { shortenBranchName } from "@/lib/branch-utils";

interface BranchCardProps {
  branch: BranchResponse;
  isSelected: boolean;
  onClick: (branch: BranchResponse) => void;
  compact?: boolean;
  className?: string;
  style?: React.CSSProperties;
}

export function BranchCard({
  branch,
  isSelected,
  onClick,
  compact = false,
  className = "",
  style,
}: BranchCardProps) {
  return (
    <button
      onClick={() => onClick(branch)}
      className={`text-left bg-card transition-all duration-200
        ${compact ? "px-2.5 py-1.5" : "px-3 py-2.5"}
        ${isSelected ? "!bg-accent z-10 relative" : "hover:!bg-muted hover:scale-[1.02] hover:shadow-md hover:-translate-y-0.5"}
        ${branch.isCurrent ? "border-l-[3px] border-l-[var(--glow-color-current)]" : ""}
        ${branch.isLocked ? "opacity-60" : ""}
        ${className}
      `}
      style={style}
    >
      <div className="flex items-center gap-2 min-w-0">
        <span className="text-sm font-medium truncate" title={branch.name}>
          {branch.commits?.[0]?.message || shortenBranchName(branch.name)}
        </span>
        {branch.isLocked && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Lock className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
            </TooltipTrigger>
            <TooltipContent side="top">
              {branch.lockReason ? `Locked: ${branch.lockReason}` : "Locked"}
            </TooltipContent>
          </Tooltip>
        )}
        {branch.needsRestack && (
          <span className="text-amber-500 shrink-0" title="Needs restack">
            &#x21BB;
          </span>
        )}
      </div>
      {!compact && (
        <div className="flex items-center gap-2 mt-1">
          {branch.pr ? (
            <PRBadge pr={branch.pr} />
          ) : (
            <span className="text-xs text-muted-foreground">no PR</span>
          )}
          <CIStatusWithTooltip ci={branch.ci} />
          <DiffStats added={branch.linesAdded} deleted={branch.linesDeleted} />
        </div>
      )}
    </button>
  );
}
