"use client";

import type { BranchResponse } from "@/lib/api";
import { PRBadge, DiffStats } from "@/components/status/status-badge";
import { CIStatusWithTooltip } from "@/components/status/ci-status";
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip";
import { GitCommitVertical, GitPullRequestDraft, Lock } from "lucide-react";
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
        ${isSelected ? "!bg-accent z-10 relative" : "hover:!bg-muted/80 hover:shadow-sm"}
        ${branch.isCurrent ? "border-l-[3px] border-l-[var(--glow-color-current)] bg-accent/30" : ""}
        ${branch.isLocked ? "opacity-60" : ""}
        ${className}
      `}
      style={style}
    >
      <div className="flex items-center gap-2 min-w-0">
        <span className="text-sm font-medium truncate" title={branch.name}>
          {branch.commits?.at(-1)?.message || shortenBranchName(branch.name)}
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
            <Tooltip>
              <TooltipTrigger asChild>
                <GitPullRequestDraft className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
              </TooltipTrigger>
              <TooltipContent side="top">No PR</TooltipContent>
            </Tooltip>
          )}
          <CIStatusWithTooltip ci={branch.ci} />
          <DiffStats added={branch.linesAdded} deleted={branch.linesDeleted} />
          {branch.commitCount > 0 && (
            <span className="flex items-center gap-0.5 text-xs text-muted-foreground" title={`${branch.commitCount} commit${branch.commitCount !== 1 ? "s" : ""}`}>
              <GitCommitVertical className="w-3 h-3" />
              {branch.commitCount}
            </span>
          )}
        </div>
      )}
    </button>
  );
}
