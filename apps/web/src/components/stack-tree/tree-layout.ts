import type { BranchResponse } from "@/lib/api";

const H_GAP = 40; // horizontal gap between siblings
const V_GAP = 80; // vertical gap between levels
const NODE_W = 160;

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
 * Computes a top-down tree layout from a flat list of branches.
 * Returns positioned nodes and edges for SVG rendering.
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

  // Build layout nodes
  const nodes: LayoutNode[] = [];
  const edges: LayoutEdge[] = [];

  for (const b of branches) {
    const pos = positions.get(b.name);
    if (!pos) continue;
    nodes.push({ branch: b, x: pos.x, y: pos.y });
  }

  // Build edges
  const nodeHeight = 56;
  for (const b of branches) {
    if (!b.parent || !byName.has(b.parent)) continue;
    const parentPos = positions.get(b.parent);
    const childPos = positions.get(b.name);
    if (!parentPos || !childPos) continue;

    edges.push({
      parentName: b.parent,
      childName: b.name,
      x1: parentPos.x,
      y1: parentPos.y + nodeHeight / 2,
      x2: childPos.x,
      y2: childPos.y - nodeHeight / 2,
      needsRestack: b.needsRestack,
    });
  }

  // Compute bounds and normalize positions.
  // Node coordinates are center-based (BranchNode translates by -NODE_W/2, -nodeHeight/2),
  // so offsets must account for half the node dimensions to keep nodes within the SVG viewport.
  const PADDING = 20;
  const allX = nodes.map((n) => n.x);
  const allY = nodes.map((n) => n.y);
  const minX = Math.min(...allX);
  const offsetX = -minX + NODE_W / 2 + PADDING;
  const offsetY = nodeHeight / 2 + PADDING;

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
  const height = Math.max(...allY) + nodeHeight + PADDING * 2;

  return { nodes, edges, width, height };
}
