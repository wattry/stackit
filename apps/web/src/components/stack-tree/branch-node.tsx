"use client";

import type { BranchResponse } from "@/lib/api";
import { PRBadge, CIStatusBadge } from "@/components/status/status-badge";

const NODE_WIDTH = 160;
const NODE_HEIGHT = 56;

interface BranchNodeProps {
  branch: BranchResponse;
  x: number;
  y: number;
  isSelected: boolean;
  onClick: (branch: BranchResponse) => void;
}

export function BranchNode({
  branch,
  x,
  y,
  isSelected,
  onClick,
}: BranchNodeProps) {
  const ciIcon = getCIIcon(branch);
  const borderClass = isSelected
    ? "stroke-blue-500 stroke-2"
    : branch.isCurrent
      ? "stroke-amber-500 stroke-2"
      : "stroke-border stroke-1";

  return (
    <g
      transform={`translate(${x - NODE_WIDTH / 2}, ${y - NODE_HEIGHT / 2})`}
      onClick={() => onClick(branch)}
      className="cursor-pointer"
    >
      <rect
        width={NODE_WIDTH}
        height={NODE_HEIGHT}
        rx={8}
        className={`fill-card ${borderClass}`}
      />

      {/* Branch name */}
      <text
        x={NODE_WIDTH / 2}
        y={20}
        textAnchor="middle"
        className="fill-foreground text-xs font-semibold"
        style={{ fontSize: 12 }}
      >
        {truncate(branch.name, 18)}
      </text>

      {/* Status line */}
      <text
        x={NODE_WIDTH / 2}
        y={38}
        textAnchor="middle"
        className="fill-muted-foreground"
        style={{ fontSize: 10 }}
      >
        {branch.pr ? `#${branch.pr.number}` : "no PR"}
        {ciIcon && ` ${ciIcon}`}
        {branch.needsRestack && " \u21BB"}
      </text>

      {/* Current branch indicator */}
      {branch.isCurrent && (
        <circle cx={NODE_WIDTH - 8} cy={8} r={4} className="fill-amber-500" />
      )}
    </g>
  );
}

export { NODE_WIDTH, NODE_HEIGHT };

function getCIIcon(branch: BranchResponse): string {
  if (!branch.ci) return "";
  switch (branch.ci.status) {
    case "passing":
      return "\u2713";
    case "failing":
      return "\u2717";
    case "pending":
      return "\u23F3";
    default:
      return "";
  }
}

function truncate(s: string, maxLen: number): string {
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen - 1) + "\u2026";
}
