"use client";

import type { BranchResponse, CIResponse } from "@/lib/api";
import { PRBadge, DiffStats } from "@/components/status/status-badge";
import { AnimatedCheckmark, AnimatedX, PulsingDot } from "@/components/ui/animated-ci-icons";
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip";
import { Lock } from "lucide-react";

interface BranchCardProps {
  branch: BranchResponse;
  isSelected: boolean;
  onClick: (branch: BranchResponse) => void;
  className?: string;
  style?: React.CSSProperties;
}

export function BranchCard({
  branch,
  isSelected,
  onClick,
  className = "",
  style,
}: BranchCardProps) {
  return (
    <button
      onClick={() => onClick(branch)}
      className={`text-left px-3 py-2.5 bg-card transition-all duration-200
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
      <div className="flex items-center gap-2 mt-1">
        {branch.pr ? (
          <PRBadge pr={branch.pr} />
        ) : (
          <span className="text-xs text-muted-foreground">no PR</span>
        )}
        <CIStatusWithTooltip ci={branch.ci} />
        <DiffStats added={branch.linesAdded} deleted={branch.linesDeleted} />
      </div>
    </button>
  );
}

/** Strip common prefixes like "user/timestamp/" to show a shorter name. */
export function shortenBranchName(name: string): string {
  const match = name.match(/^[^/]+\/\d{14}\/(.+)$/);
  return match ? match[1] : name;
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

function CheckIcon({ conclusion, status }: { conclusion: string; status: string }) {
  if (status !== "COMPLETED") {
    return <PulsingDot />;
  }
  switch (conclusion) {
    case "SUCCESS":
      return <AnimatedCheckmark />;
    case "FAILURE":
      return <AnimatedX />;
    case "NEUTRAL":
      return <span className="inline-block w-2.5 h-2.5 rounded-full bg-muted-foreground/40" />;
    default:
      return <PulsingDot />;
  }
}

function CIStatusWithTooltip({ ci }: { ci?: CIResponse }) {
  if (!ci || ci.status === "none") return null;

  const checks = ci.checks ?? [];
  const passingCount = checks.filter(
    (c) => c.status === "COMPLETED" && c.conclusion === "SUCCESS"
  ).length;
  const total = checks.length;

  const statusColorClass =
    ci.status === "passing"
      ? "text-green-600"
      : ci.status === "failing"
        ? "text-red-600"
        : "text-amber-500";

  if (total === 0) {
    return <CIStatusIcon status={ci.status} />;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`inline-flex items-center gap-1 shrink-0 ${statusColorClass}`}>
          <CIStatusIcon status={ci.status} />
          <span className="text-[10px] font-medium leading-none">
            {passingCount}/{total}
          </span>
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-64">
        <div className="space-y-1 py-0.5">
          {checks.map((check) => (
            <div key={check.name} className="flex items-center gap-2 text-xs">
              <CheckIcon conclusion={check.conclusion} status={check.status} />
              <span className="truncate">{check.name}</span>
            </div>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
