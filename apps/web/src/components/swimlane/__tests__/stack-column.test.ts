import { describe, it, expect } from "vitest";
import { hasBranching } from "../stack-column";
import { shortenBranchName } from "@/lib/branch-utils";
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

describe("hasBranching", () => {
  it("returns false for empty branches", () => {
    expect(hasBranching([])).toBe(false);
  });

  it("returns false for a single branch with no children", () => {
    expect(hasBranching([makeBranch({ name: "main" })])).toBe(false);
  });

  it("returns false for a linear stack", () => {
    const branches = [
      makeBranch({ name: "main", children: ["feat-a"] }),
      makeBranch({ name: "feat-a", parent: "main", children: ["feat-b"] }),
      makeBranch({ name: "feat-b", parent: "feat-a" }),
    ];
    expect(hasBranching(branches)).toBe(false);
  });

  it("returns true when a branch has two children", () => {
    const branches = [
      makeBranch({ name: "main", children: ["feat-a", "feat-b"] }),
      makeBranch({ name: "feat-a", parent: "main" }),
      makeBranch({ name: "feat-b", parent: "main" }),
    ];
    expect(hasBranching(branches)).toBe(true);
  });

  it("returns true when a non-root branch has multiple children", () => {
    const branches = [
      makeBranch({ name: "main", children: ["middle"] }),
      makeBranch({ name: "middle", parent: "main", children: ["left", "right"] }),
      makeBranch({ name: "left", parent: "middle" }),
      makeBranch({ name: "right", parent: "middle" }),
    ];
    expect(hasBranching(branches)).toBe(true);
  });

  it("returns false when children is undefined", () => {
    const branches = [
      makeBranch({ name: "main" }),
      makeBranch({ name: "feat", parent: "main" }),
    ];
    expect(hasBranching(branches)).toBe(false);
  });

  it("returns false when children is an empty array", () => {
    const branches = [
      makeBranch({ name: "main", children: [] }),
    ];
    expect(hasBranching(branches)).toBe(false);
  });
});

describe("shortenBranchName", () => {
  it("returns description part from user/timestamp/description pattern", () => {
    expect(shortenBranchName("jonnii/20260301202047/show-PR-titles")).toBe("show-PR-titles");
  });

  it("returns full name when no timestamp pattern", () => {
    expect(shortenBranchName("feature/my-branch")).toBe("feature/my-branch");
  });

  it("returns full name for simple branch names", () => {
    expect(shortenBranchName("main")).toBe("main");
  });

  it("handles description with slashes", () => {
    expect(shortenBranchName("user/20260301202047/feat/nested")).toBe("feat/nested");
  });
});
