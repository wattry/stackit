import type { ViewResponse, FeedEvent, BranchResponse } from "@/lib/api";

/**
 * Compares two ViewResponse snapshots and returns detected changes as FeedEvents.
 * Pure function — no side effects.
 */
export function diffViews(
  prev: ViewResponse,
  next: ViewResponse
): FeedEvent[] {
  const events: FeedEvent[] = [];
  const now = new Date().toISOString();

  // Build branch maps
  const prevBranches = new Map<string, BranchResponse>();
  const nextBranches = new Map<string, BranchResponse>();

  for (const stack of prev.stacks) {
    for (const b of stack.branches) {
      prevBranches.set(b.name, b);
    }
  }
  for (const stack of next.stacks) {
    for (const b of stack.branches) {
      nextBranches.set(b.name, b);
    }
  }

  // Branches created / deleted
  for (const name of nextBranches.keys()) {
    if (!prevBranches.has(name)) {
      events.push({ kind: "branch_created", timestamp: now, branch: name });
    }
  }
  for (const name of prevBranches.keys()) {
    if (!nextBranches.has(name)) {
      events.push({ kind: "branch_deleted", timestamp: now, branch: name });
    }
  }

  // Per-branch changes (PR state, CI, restack, revision)
  for (const [name, nextB] of nextBranches) {
    const prevB = prevBranches.get(name);
    if (!prevB) continue;

    // PR state changes
    const prevPR = prevB.pr;
    const nextPR = nextB.pr;
    if (!prevPR && nextPR) {
      events.push({
        kind: "pr_opened",
        timestamp: now,
        branch: name,
        detail: `#${nextPR.number}`,
      });
    } else if (prevPR && nextPR && prevPR.state !== nextPR.state) {
      switch (nextPR.state) {
        case "MERGED":
          events.push({
            kind: "pr_merged",
            timestamp: now,
            branch: name,
            detail: `#${nextPR.number}`,
          });
          break;
        case "CLOSED":
          events.push({
            kind: "pr_closed",
            timestamp: now,
            branch: name,
            detail: `#${nextPR.number}`,
          });
          break;
        case "OPEN":
          events.push({
            kind: "pr_opened",
            timestamp: now,
            branch: name,
            detail: `#${nextPR.number} reopened`,
          });
          break;
      }
    }

    // CI status changes
    if (prevB.ci?.status !== nextB.ci?.status && nextB.ci) {
      events.push({
        kind: "ci_changed",
        timestamp: now,
        branch: name,
        detail: nextB.ci.status,
      });
    }

    // Restack status changes
    if (!prevB.needsRestack && nextB.needsRestack) {
      events.push({ kind: "needs_restack", timestamp: now, branch: name });
    } else if (prevB.needsRestack && !nextB.needsRestack) {
      events.push({ kind: "restack_resolved", timestamp: now, branch: name });
    }

    // Revision changes
    if (prevB.revision !== nextB.revision) {
      events.push({
        kind: "revision_updated",
        timestamp: now,
        branch: name,
        detail: nextB.revision.slice(0, 7),
      });
    }
  }

  // Stack-level changes
  const prevStacks = new Set(prev.stacks.map((s) => s.rootBranch));
  const nextStacks = new Set(next.stacks.map((s) => s.rootBranch));

  for (const root of nextStacks) {
    if (!prevStacks.has(root)) {
      events.push({
        kind: "stack_created",
        timestamp: now,
        branch: root,
      });
    }
  }
  for (const root of prevStacks) {
    if (!nextStacks.has(root)) {
      events.push({
        kind: "stack_deleted",
        timestamp: now,
        branch: root,
      });
    }
  }

  // Current branch changed (client-side detection)
  if (
    prev.repo.currentBranch !== next.repo.currentBranch &&
    next.repo.currentBranch
  ) {
    events.push({
      kind: "branch_switched",
      timestamp: now,
      branch: next.repo.currentBranch,
      detail: `from ${prev.repo.currentBranch}`,
    });
  }

  return events;
}
