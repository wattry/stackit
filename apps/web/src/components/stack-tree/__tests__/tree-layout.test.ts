import { describe, it, expect } from "vitest";
import { computeTreeLayout, decomposeTree, NODE_W, COLUMN_GAP } from "../tree-layout";
import type { BranchResponse } from "@/lib/api";

function makeBranch(overrides: Partial<BranchResponse> & { name: string }): BranchResponse {
  return {
    depth: 0,
    isCurrent: false,
    needsRestack: false,
    isLocked: false,
    isFrozen: false,
    revision: "abc123",
    commitDate: "2025-01-01",
    commitAuthor: "test",
    commitCount: 1,
    linesAdded: 0,
    linesDeleted: 0,
    ...overrides,
  };
}

describe("computeTreeLayout", () => {
  it("returns empty layout for no branches", () => {
    const layout = computeTreeLayout([]);
    expect(layout.nodes).toEqual([]);
    expect(layout.edges).toEqual([]);
    expect(layout.width).toBe(0);
    expect(layout.height).toBe(0);
  });

  it("positions a single node", () => {
    const layout = computeTreeLayout([makeBranch({ name: "main" })]);
    expect(layout.nodes).toHaveLength(1);
    expect(layout.nodes[0].branch.name).toBe("main");
    expect(layout.edges).toHaveLength(0);
    expect(layout.width).toBeGreaterThan(0);
    expect(layout.height).toBeGreaterThan(0);
  });

  it("creates an edge between parent and child", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feature", parent: "main", depth: 1 }),
    ];

    const layout = computeTreeLayout(branches);
    expect(layout.nodes).toHaveLength(2);
    expect(layout.edges).toHaveLength(1);
    expect(layout.edges[0].parentName).toBe("main");
    expect(layout.edges[0].childName).toBe("feature");
  });

  it("positions child above parent (branches upward)", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feature", parent: "main", depth: 1 }),
    ];

    const layout = computeTreeLayout(branches);
    const mainNode = layout.nodes.find((n) => n.branch.name === "main")!;
    const featureNode = layout.nodes.find((n) => n.branch.name === "feature")!;
    expect(featureNode.y).toBeLessThan(mainNode.y);
  });

  it("positions siblings side by side", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feat-a", parent: "main", depth: 1 }),
      makeBranch({ name: "feat-b", parent: "main", depth: 1 }),
    ];

    const layout = computeTreeLayout(branches);
    const a = layout.nodes.find((n) => n.branch.name === "feat-a")!;
    const b = layout.nodes.find((n) => n.branch.name === "feat-b")!;

    expect(a.y).toBe(b.y);
    expect(a.x).not.toBe(b.x);
  });

  it("handles diamond topology", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "left", parent: "main", depth: 1 }),
      makeBranch({ name: "right", parent: "main", depth: 1 }),
      makeBranch({ name: "leaf", parent: "left", depth: 2 }),
    ];

    const layout = computeTreeLayout(branches);
    expect(layout.nodes).toHaveLength(4);
    expect(layout.edges).toHaveLength(3);
  });

  it("propagates needsRestack to edge", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feature", parent: "main", depth: 1, needsRestack: true }),
    ];

    const layout = computeTreeLayout(branches);
    expect(layout.edges[0].needsRestack).toBe(true);
  });

  it("edge has needsRestack false when branch does not need restack", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feature", parent: "main", depth: 1, needsRestack: false }),
    ];

    const layout = computeTreeLayout(branches);
    expect(layout.edges[0].needsRestack).toBe(false);
  });
});

describe("decomposeTree", () => {
  it("returns empty segment for no branches", () => {
    const segment = decomposeTree([]);
    expect(segment.branches).toEqual([]);
    expect(segment.forks).toBeUndefined();
    expect(segment.width).toBe(NODE_W);
  });

  it("returns a single linear segment for a chain", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "a", parent: "main", depth: 1 }),
      makeBranch({ name: "b", parent: "a", depth: 2 }),
    ];

    const segment = decomposeTree(branches);
    expect(segment.branches.map((b) => b.name)).toEqual(["main", "a", "b"]);
    expect(segment.forks).toBeUndefined();
    expect(segment.width).toBe(NODE_W);
  });

  it("splits at fork point", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "left", parent: "main", depth: 1 }),
      makeBranch({ name: "right", parent: "main", depth: 1 }),
    ];

    const segment = decomposeTree(branches);
    expect(segment.branches.map((b) => b.name)).toEqual(["main"]);
    expect(segment.forks).toHaveLength(2);
    expect(segment.forks![0].branches.map((b) => b.name)).toEqual(["left"]);
    expect(segment.forks![1].branches.map((b) => b.name)).toEqual(["right"]);
  });

  it("computes width for two-way fork", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "left", parent: "main", depth: 1 }),
      makeBranch({ name: "right", parent: "main", depth: 1 }),
    ];

    const segment = decomposeTree(branches);
    expect(segment.width).toBe(NODE_W * 2 + COLUMN_GAP);
  });

  it("handles linear chain above fork", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "a", parent: "main", depth: 1 }),
      makeBranch({ name: "left", parent: "a", depth: 2 }),
      makeBranch({ name: "right", parent: "a", depth: 2 }),
    ];

    const segment = decomposeTree(branches);
    // trunk chain includes main and a (the fork point)
    expect(segment.branches.map((b) => b.name)).toEqual(["main", "a"]);
    expect(segment.forks).toHaveLength(2);
  });

  it("handles nested forks", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "a", parent: "main", depth: 1 }),
      makeBranch({ name: "b", parent: "a", depth: 2 }),
      makeBranch({ name: "b1", parent: "b", depth: 3 }),
      makeBranch({ name: "b2", parent: "b", depth: 3 }),
      makeBranch({ name: "c", parent: "a", depth: 2 }),
    ];

    const segment = decomposeTree(branches);
    expect(segment.branches.map((b) => b.name)).toEqual(["main", "a"]);
    expect(segment.forks).toHaveLength(2);

    // First fork: b with nested forks b1, b2
    const bFork = segment.forks![0];
    expect(bFork.branches.map((b) => b.name)).toEqual(["b"]);
    expect(bFork.forks).toHaveLength(2);
    expect(bFork.forks![0].branches[0].name).toBe("b1");
    expect(bFork.forks![1].branches[0].name).toBe("b2");

    // Second fork: c (linear)
    expect(segment.forks![1].branches.map((b) => b.name)).toEqual(["c"]);
    expect(segment.forks![1].forks).toBeUndefined();
  });

  it("handles linear chain after fork", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "left", parent: "main", depth: 1 }),
      makeBranch({ name: "left-child", parent: "left", depth: 2 }),
      makeBranch({ name: "right", parent: "main", depth: 1 }),
    ];

    const segment = decomposeTree(branches);
    expect(segment.forks).toHaveLength(2);
    // Left fork includes the chain left → left-child
    expect(segment.forks![0].branches.map((b) => b.name)).toEqual([
      "left",
      "left-child",
    ]);
    expect(segment.forks![1].branches.map((b) => b.name)).toEqual(["right"]);
  });

  it("nested fork width accounts for grandchildren", () => {
    const branches = [
      makeBranch({ name: "root" }),
      makeBranch({ name: "a", parent: "root", depth: 1 }),
      makeBranch({ name: "a1", parent: "a", depth: 2 }),
      makeBranch({ name: "a2", parent: "a", depth: 2 }),
      makeBranch({ name: "b", parent: "root", depth: 1 }),
    ];

    const segment = decomposeTree(branches);
    // Fork a: width = 2*NODE_W + COLUMN_GAP (for a1 and a2)
    // Fork b: width = NODE_W
    // Total: (2*NODE_W + COLUMN_GAP) + COLUMN_GAP + NODE_W
    const expectedWidth = 3 * NODE_W + 2 * COLUMN_GAP;
    expect(segment.width).toBe(expectedWidth);
  });
});
