"use client";

import { useCallback, useMemo, useState } from "react";
import { useRepo } from "@/components/providers/repo-provider";
import { OwnerSwimlane, getLastActiveDate } from "@/components/owner-swimlane";
import { BranchDetail } from "@/components/branch-detail/branch-detail";
import { StackDetailPanel } from "@/components/branch-detail/stack-detail";
import { DetailEmptyState } from "@/components/branch-detail/detail-empty-state";
import { EventFeed } from "@/components/event-feed";
import { Separator } from "@/components/ui/separator";
import { RecentlyMerged } from "@/components/recently-merged";
import { BackgroundMesh } from "@/components/ui/background-mesh";
import { SkeletonSwimlane } from "@/components/ui/skeleton-shimmer";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import type { BranchResponse, StackDetail } from "@/lib/api";
import { formatTimeAgo } from "@/lib/time";

type Selection =
  | { type: "branch"; name: string }
  | { type: "stack"; rootBranch: string };

export default function Home() {
  const {
    repo,
    stackDetails,
    recentlyMerged,
    loading,
    error,
    lastUpdated,
    refresh,
  } = useRepo();
  const [selection, setSelection] = useState<Selection | null>(() => {
    if (typeof window === "undefined") return null;
    const params = new URLSearchParams(window.location.search);
    const branch = params.get("branch");
    if (branch) return { type: "branch", name: branch };
    const stack = params.get("stack");
    if (stack) return { type: "stack", rootBranch: stack };
    return null;
  });

  const selectedBranch = useMemo(() => {
    if (!selection || selection.type !== "branch") return null;
    for (const stack of stackDetails) {
      const found = stack.branches.find((b) => b.name === selection.name);
      if (found) return found;
    }
    return null;
  }, [selection, stackDetails]);

  const selectedStack = useMemo(() => {
    if (!selection || selection.type !== "stack") return null;
    return stackDetails.find((s) => s.rootBranch === selection.rootBranch) ?? null;
  }, [selection, stackDetails]);

  const handleSelectBranch = useCallback((branch: BranchResponse | null) => {
    const url = new URL(window.location.href);
    url.searchParams.delete("stack");
    if (branch) {
      setSelection({ type: "branch", name: branch.name });
      url.searchParams.set("branch", branch.name);
    } else {
      setSelection(null);
      url.searchParams.delete("branch");
    }
    window.history.replaceState({}, "", url.toString());
  }, []);

  const handleSelectStack = useCallback((stack: StackDetail) => {
    setSelection((prev) => {
      const deselecting = prev?.type === "stack" && prev.rootBranch === stack.rootBranch;
      const next = deselecting ? null : { type: "stack" as const, rootBranch: stack.rootBranch };

      queueMicrotask(() => {
        const url = new URL(window.location.href);
        url.searchParams.delete("branch");
        if (next) {
          url.searchParams.set("stack", stack.rootBranch);
        } else {
          url.searchParams.delete("stack");
        }
        window.history.replaceState({}, "", url.toString());
      });

      return next;
    });
  }, []);

  const handleNavigateToBranch = useCallback(
    (name: string) => {
      for (const stack of stackDetails) {
        const found = stack.branches.find((b) => b.name === name);
        if (found) {
          handleSelectBranch(found);
          return;
        }
      }
    },
    [stackDetails, handleSelectBranch]
  );

  const handleStackBranchSelect = useCallback(
    (branch: BranchResponse) => {
      handleSelectBranch(branch);
    },
    [handleSelectBranch]
  );

  const currentUser = repo?.currentUser;

  const { yourStacks, otherOwners } = useMemo(() => {
    const yours: StackDetail[] = [];
    const others = new Map<string, StackDetail[]>();
    for (const stack of stackDetails) {
      if (!stack.owner || stack.owner === currentUser) {
        yours.push(stack);
      } else {
        const existing = others.get(stack.owner);
        if (existing) {
          existing.push(stack);
        } else {
          others.set(stack.owner, [stack]);
        }
      }
    }
    const sortedOthers = [...others.entries()].sort(([a], [b]) =>
      a.localeCompare(b)
    );
    return { yourStacks: yours, otherOwners: sortedOthers };
  }, [stackDetails, currentUser]);

  const hasSelection = selectedBranch || selectedStack;

  if (loading) {
    return (
      <>
        <BackgroundMesh />
        <SkeletonSwimlane />
      </>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-screen gap-4">
        <p className="text-destructive">{error}</p>
        <p className="text-sm text-muted-foreground">
          Make sure stackit-web is running on{" "}
          {process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}
        </p>
        <button
          onClick={refresh}
          className="text-sm underline text-muted-foreground hover:text-foreground"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen">
      <BackgroundMesh />
      {/* Header */}
      <header className="relative flex items-center justify-between px-4 py-2 border-b shrink-0">
        <div className="flex items-center gap-3">
          <span className="font-semibold text-sm">stackit</span>
          {repo && (
            <span className="text-sm text-muted-foreground font-mono">
              {repo.owner}/{repo.repo}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-xs text-muted-foreground">
              {formatTimeAgo(lastUpdated)}
            </span>
          )}
          <ThemeToggle />
          <button
            onClick={refresh}
            className="text-xs text-muted-foreground hover:text-foreground"
            title="Refresh"
          >
            &#x21BB;
          </button>
        </div>
        {/* Animated gradient accent bar */}
        <div
          className="absolute bottom-0 left-0 right-0 h-0.5 animate-gradient-shift"
          style={{
            background: "linear-gradient(90deg, var(--gradient-start), var(--gradient-mid), var(--gradient-end), var(--gradient-start))",
          }}
        />
      </header>

      {/* Main content: stacks area + detail panel */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 overflow-y-auto overflow-x-hidden">
          {stackDetails.length > 0 ? (
            <div className="flex flex-col justify-end min-h-full">
              {/* Swimlanes: only this area scrolls horizontally */}
              <div className="overflow-x-auto">
                <div className="flex items-end gap-8 p-6 pb-4 min-w-max">
                  {/* Your stacks */}
                  {yourStacks.length > 0 && (
                    <OwnerSwimlane
                      label="You"
                      stacks={yourStacks}
                      selectedBranch={selection?.type === "branch" ? selection.name : null}
                      selectedStack={selection?.type === "stack" ? selection.rootBranch : null}
                      onSelectBranch={handleSelectBranch}
                      onSelectStack={handleSelectStack}
                    />
                  )}

                  {/* Teammate swimlanes */}
                  {otherOwners.map(([owner, stacks]) => (
                    <OwnerSwimlane
                      key={owner}
                      label={`@${owner}`}
                      lastActive={getLastActiveDate(stacks)}
                      stacks={stacks}
                      selectedBranch={selection?.type === "branch" ? selection.name : null}
                      selectedStack={selection?.type === "stack" ? selection.rootBranch : null}
                      onSelectBranch={handleSelectBranch}
                      onSelectStack={handleSelectStack}
                    />
                  ))}
                </div>
              </div>

              {/* Trunk line */}
              <div className="flex items-center gap-2 px-6 pb-2 shrink-0">
                <div className="flex-1 border-t-2 border-dashed border-muted-foreground/30" />
                <span className="text-xs font-mono text-muted-foreground">{repo?.trunk}</span>
                <div className="flex-1 border-t-2 border-dashed border-muted-foreground/30" />
              </div>

              {/* Recent trunk commits */}
              <RecentlyMerged />
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
              No stacks found
            </div>
          )}
        </div>

        {/* Right: detail + event feed panel (always visible) */}
        <div className="flex shrink-0">
          <Separator orientation="vertical" />
          <div className="w-[480px] shrink-0 flex flex-col overflow-hidden">
            {hasSelection ? (
              <div className="flex-1 overflow-auto p-4">
                {selectedBranch && (
                  <BranchDetail
                    branch={selectedBranch}
                    onNavigateToBranch={handleNavigateToBranch}
                  />
                )}
                {selectedStack && (
                  <StackDetailPanel
                    stack={selectedStack}
                    onSelectBranch={handleStackBranchSelect}
                  />
                )}
              </div>
            ) : (
              <div className="flex-1">
                <DetailEmptyState />
              </div>
            )}
            <Separator />
            <div className="p-3 overflow-auto">
              <EventFeed />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
