"use client";

interface BranchEdgeProps {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  needsRestack?: boolean;
}

export function BranchEdge({ x1, y1, x2, y2, needsRestack }: BranchEdgeProps) {
  // Vertical path with a slight curve at the bend
  const midY = y1 + (y2 - y1) / 2;

  const path =
    x1 === x2
      ? `M ${x1} ${y1} L ${x2} ${y2}`
      : `M ${x1} ${y1} L ${x1} ${midY} Q ${x1} ${midY + 10} ${x1 + (x2 - x1) * 0.2} ${midY + 10} L ${x2 - (x2 - x1) * 0.2} ${midY + 10} Q ${x2} ${midY + 10} ${x2} ${midY + 20} L ${x2} ${y2}`;

  return (
    <path
      d={path}
      fill="none"
      className={needsRestack ? "stroke-amber-400" : "stroke-muted-foreground/40"}
      strokeWidth={needsRestack ? 2 : 1.5}
      strokeDasharray={needsRestack ? "4 3" : undefined}
    />
  );
}
