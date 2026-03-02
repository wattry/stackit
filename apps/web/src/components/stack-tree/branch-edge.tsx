"use client";

interface BranchEdgeProps {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  needsRestack?: boolean;
}

export function BranchEdge({ x1, y1, x2, y2, needsRestack }: BranchEdgeProps) {
  // Vertical path with a slight curve at the bend (works for both up and down directions)
  const midY = y1 + (y2 - y1) / 2;
  const dy = Math.sign(y2 - y1) || 1;
  const r = dy * 10;

  const path =
    x1 === x2
      ? `M ${x1} ${y1} L ${x2} ${y2}`
      : `M ${x1} ${y1} L ${x1} ${midY} Q ${x1} ${midY + r} ${x1 + (x2 - x1) * 0.2} ${midY + r} L ${x2 - (x2 - x1) * 0.2} ${midY + r} Q ${x2} ${midY + r} ${x2} ${midY + r * 2} L ${x2} ${y2}`;

  return (
    <path
      d={path}
      fill="none"
      className={needsRestack ? "stroke-amber-400" : "stroke-muted-foreground/40"}
      strokeWidth={needsRestack ? 2 : 1.5}
    />
  );
}
