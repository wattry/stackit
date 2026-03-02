"use client";

import { useMemo } from "react";
import type { BranchResponse } from "@/lib/api";
import { BranchCard } from "@/components/branch-card";
import { decomposeTree, type TreeSegment, NODE_W, COLUMN_GAP } from "./tree-layout";

interface SegmentTreeProps {
  branches: BranchResponse[];
  selectedBranch: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
  footer?: React.ReactNode;
}

export function SegmentTree({
  branches,
  selectedBranch,
  onSelectBranch,
  footer,
}: SegmentTreeProps) {
  const segment = useMemo(() => decomposeTree(branches), [branches]);

  if (segment.branches.length === 0 && !segment.forks?.length) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        No branches in this stack
      </div>
    );
  }

  return (
    <SegmentNode
      segment={segment}
      selectedBranch={selectedBranch}
      onSelectBranch={onSelectBranch}
      footer={footer}
    />
  );
}

interface SegmentNodeProps {
  segment: TreeSegment;
  selectedBranch: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
  footer?: React.ReactNode;
}

function SegmentNode({
  segment,
  selectedBranch,
  onSelectBranch,
  footer,
}: SegmentNodeProps) {
  const hasForks = segment.forks && segment.forks.length > 0;
  // Display order: leaf at top (reverse of root-to-leaf)
  const displayBranches = [...segment.branches].reverse();

  return (
    <div
      className="flex flex-col items-center"
      style={{ width: segment.width }}
    >
      {/* Fork children above */}
      {hasForks && (
        <>
          <div
            className="flex items-end"
            style={{ gap: COLUMN_GAP }}
          >
            {segment.forks!.map((fork, i) => (
              <SegmentNode
                key={fork.branches[0]?.name ?? i}
                segment={fork}
                selectedBranch={selectedBranch}
                onSelectBranch={onSelectBranch}
              />
            ))}
          </div>
          <ForkConnector segment={segment} />
        </>
      )}

      {/* Linear trunk segment */}
      {displayBranches.length > 0 && (
        <div className="flex flex-col" style={{ width: NODE_W }}>
          <div className="flex flex-col bg-card rounded-lg">
            {displayBranches.map((branch, i) => {
              const isFirst = i === 0;
              const isLast = i === displayBranches.length - 1;

              return (
                <BranchCard
                  key={branch.name}
                  branch={branch}
                  isSelected={selectedBranch === branch.name}
                  onClick={onSelectBranch}
                  className={`border-x border-t
                    ${isLast ? "border-b rounded-b-lg relative z-[1] shadow-[0_4px_6px_-2px_rgba(0,0,0,0.15)]" : ""}
                    ${isFirst ? "rounded-t-lg" : ""}
                  `}
                />
              );
            })}
          </div>
          {footer}
        </div>
      )}
    </div>
  );
}

const CONNECTOR_H = 28;
const CURVE_R = 6;

function ForkConnector({ segment }: { segment: TreeSegment }) {
  if (!segment.forks || segment.forks.length < 2) return null;

  const totalWidth = segment.width;
  const forks = segment.forks;

  const forksWidth =
    forks.reduce((sum, f) => sum + f.width, 0) +
    COLUMN_GAP * (forks.length - 1);

  // Center the forks area within the total width
  const startX = (totalWidth - forksWidth) / 2;

  // Compute center X of each fork column
  const centers: number[] = [];
  let x = startX;
  for (const fork of forks) {
    centers.push(x + fork.width / 2);
    x += fork.width + COLUMN_GAP;
  }

  // Trunk center (where all forks converge)
  const trunkX = totalWidth / 2;
  const midY = CONNECTOR_H / 2;
  const leftX = Math.min(...centers);
  const rightX = Math.max(...centers);

  return (
    <svg
      width={totalWidth}
      height={CONNECTOR_H}
      className="shrink-0"
      data-testid="fork-connector"
    >
      {/* Vertical stubs from each fork column down to the horizontal line */}
      {centers.map((cx, i) => (
        <line
          key={`v-${i}`}
          x1={cx}
          y1={0}
          x2={cx}
          y2={midY}
          className={
            forks[i].branches[0]?.needsRestack
              ? "stroke-amber-400"
              : "stroke-muted-foreground/40"
          }
          strokeWidth={forks[i].branches[0]?.needsRestack ? 2 : 1.5}
        />
      ))}
      {/* Horizontal line connecting all fork columns */}
      <line
        x1={leftX}
        y1={midY}
        x2={rightX}
        y2={midY}
        className="stroke-muted-foreground/40"
        strokeWidth={1.5}
      />
      {/* Vertical line from center down to trunk */}
      <line
        x1={trunkX}
        y1={midY}
        x2={trunkX}
        y2={CONNECTOR_H}
        className="stroke-muted-foreground/40"
        strokeWidth={1.5}
      />
    </svg>
  );
}
