import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { OwnerSwimlane } from "../owner-swimlane";
import type { BranchResponse, StackDetail } from "@/lib/api";

// Mock motion/react to avoid animation issues in tests
vi.mock("motion/react", () => ({
  motion: {
    div: ({ children, ...props }: React.PropsWithChildren<Record<string, unknown>>) => <div {...props}>{children}</div>,
    button: ({ children, ...props }: React.PropsWithChildren<Record<string, unknown>>) => <button {...props}>{children}</button>,
  },
  AnimatePresence: ({ children }: React.PropsWithChildren) => <>{children}</>,
}));

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

function makeStack(overrides: Partial<StackDetail> & { rootBranch: string; branches: BranchResponse[] }): StackDetail {
  return {
    title: "",
    status: "shippable",
    branchCount: overrides.branches.length,
    prCount: 0,
    isCurrent: false,
    ...overrides,
  };
}

const noop = () => {};

describe("OwnerSwimlane", () => {
  it("renders linear stack without tree edges SVG", () => {
    const stack = makeStack({
      rootBranch: "main",
      branches: [
        makeBranch({ name: "main", children: ["feat"] }),
        makeBranch({ name: "feat", parent: "main" }),
      ],
    });

    const { container } = render(
      <OwnerSwimlane
        label="me"
        stacks={[stack]}
        selectedBranch={null}
        selectedStack={null}
        onSelectBranch={noop}
        onSelectStack={noop}
      />
    );

    // Linear stacks render StackColumn, not StackTree
    expect(container.querySelector('[data-testid="stack-tree-edges"]')).toBeNull();
    expect(screen.getAllByRole("button").length).toBeGreaterThan(0);
  });

  it("renders branching stack with fork connector", () => {
    const stack = makeStack({
      rootBranch: "main",
      branches: [
        makeBranch({ name: "main", children: ["left", "right"] }),
        makeBranch({ name: "left", parent: "main" }),
        makeBranch({ name: "right", parent: "main" }),
      ],
    });

    const { container } = render(
      <OwnerSwimlane
        label="me"
        stacks={[stack]}
        selectedBranch={null}
        selectedStack={null}
        onSelectBranch={noop}
        onSelectStack={noop}
      />
    );

    // Branching stacks render SegmentTree which uses fork connectors at branch points
    expect(container.querySelector('[data-testid="fork-connector"]')).not.toBeNull();
  });

  it("renders mixed stacks with correct visualization for each", () => {
    const linearStack = makeStack({
      rootBranch: "linear-root",
      branches: [
        makeBranch({ name: "linear-root", children: ["child"] }),
        makeBranch({ name: "child", parent: "linear-root" }),
      ],
    });

    const branchingStack = makeStack({
      rootBranch: "branching-root",
      branches: [
        makeBranch({ name: "branching-root", children: ["left", "right"] }),
        makeBranch({ name: "left", parent: "branching-root" }),
        makeBranch({ name: "right", parent: "branching-root" }),
      ],
    });

    const { container } = render(
      <OwnerSwimlane
        label="me"
        stacks={[linearStack, branchingStack]}
        selectedBranch={null}
        selectedStack={null}
        onSelectBranch={noop}
        onSelectStack={noop}
      />
    );

    // Exactly one fork connector SVG (from the branching stack)
    const forkConnectors = container.querySelectorAll('[data-testid="fork-connector"]');
    expect(forkConnectors.length).toBe(1);
  });

  it("renders branch cards with same content in both views", () => {
    const linearStack = makeStack({
      rootBranch: "lin",
      branches: [
        makeBranch({ name: "lin", children: ["lin-child"] }),
        makeBranch({ name: "lin-child", parent: "lin" }),
      ],
    });

    const branchingStack = makeStack({
      rootBranch: "br",
      branches: [
        makeBranch({ name: "br", children: ["br-left", "br-right"] }),
        makeBranch({ name: "br-left", parent: "br" }),
        makeBranch({ name: "br-right", parent: "br" }),
      ],
    });

    render(
      <OwnerSwimlane
        label="me"
        stacks={[linearStack, branchingStack]}
        selectedBranch={null}
        selectedStack={null}
        onSelectBranch={noop}
        onSelectStack={noop}
      />
    );

    // Both views show "no PR" text for branches without PRs (shared BranchCard)
    const noPrLabels = screen.getAllByText("no PR");
    // 5 branches total across both stacks
    expect(noPrLabels.length).toBe(5);
  });
});
