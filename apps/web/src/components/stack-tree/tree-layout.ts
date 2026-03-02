import type { BranchResponse } from "@/lib/api";

export const H_GAP = 24; // horizontal gap between siblings
export const V_GAP = 88; // vertical gap between levels
export const NODE_W = 256;
export const NODE_H = 64;
export const TREE_PADDING = 20;
export const COLUMN_GAP = 12; // gap between fork columns

export interface LayoutNode {
  branch: BranchResponse;
  x: number;
  y: number;
}

export interface LayoutEdge {
  parentName: string;
  childName: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  needsRestack: boolean;
}

export interface TreeLayout {
  nodes: LayoutNode[];
  edges: LayoutEdge[];
  width: number;
  height: number;
}

/**
 * Computes a bottom-up tree layout from a flat list of branches.
 * Root is at the bottom, leaves branch upward (matching the stacking metaphor).
 */
export function computeTreeLayout(branches: BranchResponse[]): TreeLayout {
  if (branches.length === 0) {
    return { nodes: [], edges: [], width: 0, height: 0 };
  }

  const byName = new Map<string, BranchResponse>();
  const childrenOf = new Map<string, string[]>();

  for (const b of branches) {
    byName.set(b.name, b);
  }

  // Build children map
  for (const b of branches) {
    if (b.parent && byName.has(b.parent)) {
      const existing = childrenOf.get(b.parent) || [];
      existing.push(b.name);
      childrenOf.set(b.parent, existing);
    }
  }

  // Find roots (branches whose parent is not in the branch list)
  const roots = branches.filter(
    (b) => !b.parent || !byName.has(b.parent)
  );

  // Assign positions using a recursive layout
  const positions = new Map<string, { x: number; y: number }>();
  let nextX = 0;

  function layout(name: string, depth: number): { minX: number; maxX: number } {
    const children = childrenOf.get(name) || [];

    if (children.length === 0) {
      // Leaf node
      const x = nextX;
      nextX += NODE_W + H_GAP;
      positions.set(name, { x, y: depth * V_GAP });
      return { minX: x, maxX: x };
    }

    // Layout children first
    const childRanges = children.map((child) => layout(child, depth + 1));

    // Center parent above children
    const minX = Math.min(...childRanges.map((r) => r.minX));
    const maxX = Math.max(...childRanges.map((r) => r.maxX));
    const x = (minX + maxX) / 2;

    positions.set(name, { x, y: depth * V_GAP });
    return { minX: Math.min(minX, x), maxX: Math.max(maxX, x) };
  }

  for (const root of roots) {
    layout(root.name, 0);
  }

  // Flip Y so root is at the bottom and leaves branch upward
  const maxY = Math.max(...Array.from(positions.values()).map((p) => p.y));
  for (const [name, pos] of positions) {
    positions.set(name, { x: pos.x, y: maxY - pos.y });
  }

  // Build layout nodes
  const nodes: LayoutNode[] = [];
  const edges: LayoutEdge[] = [];

  for (const b of branches) {
    const pos = positions.get(b.name);
    if (!pos) continue;
    nodes.push({ branch: b, x: pos.x, y: pos.y });
  }

  // Build edges (parent is below child after flip)
  for (const b of branches) {
    if (!b.parent || !byName.has(b.parent)) continue;
    const parentPos = positions.get(b.parent);
    const childPos = positions.get(b.name);
    if (!parentPos || !childPos) continue;

    edges.push({
      parentName: b.parent,
      childName: b.name,
      x1: parentPos.x,
      y1: parentPos.y - NODE_H / 2,
      x2: childPos.x,
      y2: childPos.y + NODE_H / 2,
      needsRestack: b.needsRestack,
    });
  }

  // Compute bounds and normalize positions.
  // Node coordinates are center-based, so offsets must account for
  // half the node dimensions to keep nodes within the viewport.
  const PADDING = TREE_PADDING;
  const allX = nodes.map((n) => n.x);
  const allY = nodes.map((n) => n.y);
  const minX = Math.min(...allX);
  const offsetX = -minX + NODE_W / 2 + PADDING;
  const offsetY = NODE_H / 2 + PADDING;

  for (const node of nodes) {
    node.x += offsetX;
    node.y += offsetY;
  }
  for (const edge of edges) {
    edge.x1 += offsetX;
    edge.x2 += offsetX;
    edge.y1 += offsetY;
    edge.y2 += offsetY;
  }

  const width = (Math.max(...allX) - minX) + NODE_W + PADDING * 2;
  const height = Math.max(...allY) + NODE_H + PADDING * 2;

  return { nodes, edges, width, height };
}

/**
 * A segment of a stack tree: a linear chain of branches that may fork
 * at the top into multiple child segments.
 */
export interface TreeSegment {
  /** Branches forming a linear chain, ordered root-to-leaf (bottom to top in display). */
  branches: BranchResponse[];
  /** Child segments if the topmost branch has multiple children. */
  forks?: TreeSegment[];
  /** Computed width in pixels for layout. */
  width: number;
}

/**
 * Decomposes a flat branch list into a tree of linear segments.
 * Each segment is a chain of single-child branches. When a branch has
 * multiple children, the segment records them as `forks`.
 */
export function decomposeTree(branches: BranchResponse[]): TreeSegment {
  if (branches.length === 0) {
    return { branches: [], width: NODE_W };
  }

  const byName = new Map<string, BranchResponse>();
  const childrenOf = new Map<string, string[]>();

  for (const b of branches) {
    byName.set(b.name, b);
  }

  // Build children map preserving DFS order from the branch list
  for (const b of branches) {
    if (b.parent && byName.has(b.parent)) {
      const existing = childrenOf.get(b.parent) || [];
      existing.push(b.name);
      childrenOf.set(b.parent, existing);
    }
  }

  const roots = branches.filter(
    (b) => !b.parent || !byName.has(b.parent)
  );

  function buildSegment(startName: string): TreeSegment {
    const chain: BranchResponse[] = [];
    let current = startName;

    for (;;) {
      const branch = byName.get(current);
      if (!branch) break;
      chain.push(branch);

      const children = childrenOf.get(current) || [];
      if (children.length === 1) {
        current = children[0];
      } else if (children.length > 1) {
        const forks = children.map((child) => buildSegment(child));
        const forksWidth =
          forks.reduce((sum, f) => sum + f.width, 0) +
          COLUMN_GAP * (forks.length - 1);
        return {
          branches: chain,
          forks,
          width: Math.max(NODE_W, forksWidth),
        };
      } else {
        break;
      }
    }

    return { branches: chain, width: NODE_W };
  }

  if (roots.length === 1) {
    return buildSegment(roots[0].name);
  }

  // Multiple roots: treat as parallel forks
  const forks = roots.map((r) => buildSegment(r.name));
  const forksWidth =
    forks.reduce((sum, f) => sum + f.width, 0) +
    COLUMN_GAP * (forks.length - 1);
  return {
    branches: [],
    forks,
    width: Math.max(NODE_W, forksWidth),
  };
}
