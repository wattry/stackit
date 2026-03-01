import { describe, it, expect } from "vitest";
import { computeTreeLayout } from "../tree-layout";
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

  it("positions child below parent", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feature", parent: "main", depth: 1 }),
    ];

    const layout = computeTreeLayout(branches);
    const mainNode = layout.nodes.find((n) => n.branch.name === "main")!;
    const featureNode = layout.nodes.find((n) => n.branch.name === "feature")!;
    expect(featureNode.y).toBeGreaterThan(mainNode.y);
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
