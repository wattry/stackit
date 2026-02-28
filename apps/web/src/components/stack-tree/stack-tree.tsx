"use client";

import { useMemo } from "react";
import type { BranchResponse } from "@/lib/api";
import { BranchNode, NODE_HEIGHT } from "./branch-node";
import { BranchEdge } from "./branch-edge";
import { computeTreeLayout } from "./tree-layout";

interface StackTreeProps {
  branches: BranchResponse[];
  selectedBranch: string | null;
  onSelectBranch: (branch: BranchResponse) => void;
}

export function StackTree({
  branches,
  selectedBranch,
  onSelectBranch,
}: StackTreeProps) {
  const layout = useMemo(() => computeTreeLayout(branches), [branches]);

  if (layout.nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        No branches in this stack
      </div>
    );
  }

  return (
    <div className="overflow-auto flex-1">
      <svg
        width={layout.width}
        height={layout.height}
        className="mx-auto"
        style={{ minWidth: layout.width, minHeight: layout.height }}
      >
        {/* Edges first (below nodes) */}
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

        {/* Nodes */}
        {layout.nodes.map((node) => (
          <BranchNode
            key={node.branch.name}
            branch={node.branch}
            x={node.x}
            y={node.y}
            isSelected={node.branch.name === selectedBranch}
            onClick={onSelectBranch}
          />
        ))}
      </svg>
    </div>
  );
}
