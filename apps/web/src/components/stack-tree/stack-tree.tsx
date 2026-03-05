"use client";

import { useMemo } from "react";
import type { BranchResponse } from "@/lib/api";
import { BranchCard } from "@/components/swimlane/branch-card";
import { BranchEdge } from "./branch-edge";
import { computeTreeLayout, NODE_W, NODE_H, TREE_PADDING } from "./tree-layout";

const FOOTER_H = 28;

interface StackTreeProps {
  branches: BranchResponse[];
  selectedBranch: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
  footer?: React.ReactNode;
}

export function StackTree({
  branches,
  selectedBranch,
  onSelectBranch,
  footer,
}: StackTreeProps) {
  const layout = useMemo(() => computeTreeLayout(branches), [branches]);

  if (layout.nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        No branches in this stack
      </div>
    );
  }

  // Root node is at the bottom (highest Y after flip)
  const rootNode = layout.nodes.reduce((a, b) => (a.y > b.y ? a : b));

  // When footer is present, remove bottom padding so footer sits flush at the bottom
  const totalHeight = footer
    ? layout.height - TREE_PADDING + FOOTER_H
    : layout.height;

  return (
    <div className="overflow-auto flex-1">
      <div
        className="relative mx-auto"
        style={{ width: layout.width, height: totalHeight }}
      >
        {/* SVG edges layer (behind cards) */}
        <svg
          className="absolute inset-0 pointer-events-none"
          width={layout.width}
          height={totalHeight}
          data-testid="stack-tree-edges"
        >
          {layout.edges.map((edge) => (
            <BranchEdge
              key={`${edge.parentName}-${edge.childName}`}
              x1={edge.x1}
              y1={edge.y1}
              x2={edge.x2}
              y2={edge.y2}
              needsRestack={edge.needsRestack}
            />
          ))}
        </svg>

        {/* HTML branch cards layer */}
        {layout.nodes.map((node) => {
          const isRoot = node === rootNode;
          return (
            <div
              key={node.branch.name}
              className="absolute flex flex-col"
              style={{
                left: node.x - NODE_W / 2,
                top: node.y - NODE_H / 2,
                width: NODE_W,
              }}
            >
              <BranchCard
                branch={node.branch}
                isSelected={node.branch.name === selectedBranch}
                onClick={onSelectBranch}
                className={
                  isRoot && footer
                    ? "rounded-t-lg border-x border-t shadow-sm w-full"
                    : "rounded-lg border shadow-sm w-full"
                }
              />
              {isRoot && footer}
            </div>
          );
        })}
      </div>
    </div>
  );
}
