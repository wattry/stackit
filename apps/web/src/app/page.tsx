"use client";

import { useCallback, useMemo, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { useRepo } from "@/components/providers/repo-provider";
import { OwnerSwimlane, getLastActiveDate } from "@/components/owner-swimlane";
import { BranchDetail } from "@/components/branch-detail/branch-detail";
import { RecentlyMerged } from "@/components/recently-merged";
import { Separator } from "@/components/ui/separator";
import { BackgroundMesh } from "@/components/ui/background-mesh";
import { SkeletonSwimlane } from "@/components/ui/skeleton-shimmer";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import type { BranchResponse, StackDetail } from "@/lib/api";
import { formatTimeAgo } from "@/lib/time";

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
  const [selectedBranchName, setSelectedBranchName] = useState<string | null>(
    () => {
      if (typeof window === "undefined") return null;
      return new URLSearchParams(window.location.search).get("branch");
    }
  );

  const selectedBranch = useMemo(() => {
    if (!selectedBranchName) return null;
    for (const stack of stackDetails) {
      const found = stack.branches.find((b) => b.name === selectedBranchName);
      if (found) return found;
    }
    return null;
  }, [selectedBranchName, stackDetails]);

  const handleSelectBranch = useCallback((branch: BranchResponse | null) => {
    setSelectedBranchName(branch?.name ?? null);
    const url = new URL(window.location.href);
    if (branch) {
      url.searchParams.set("branch", branch.name);
    } else {
      url.searchParams.delete("branch");
    }
    window.history.replaceState({}, "", url.toString());
  }, []);

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
        <div className="flex-1 overflow-auto">
          {stackDetails.length > 0 ? (
            <div className="flex flex-col justify-end min-h-full">
              {/* All swimlanes in one horizontal row */}
              <div className="flex items-end gap-8 p-6 pb-4 min-w-max">
                {/* Your stacks */}
                {yourStacks.length > 0 && (
                  <OwnerSwimlane
                    label="You"
                    stacks={yourStacks}
                    selectedBranch={selectedBranch?.name ?? null}
                    onSelectBranch={handleSelectBranch}
                  />
                )}

                {/* Teammate swimlanes */}
                {otherOwners.map(([owner, stacks]) => (
                  <OwnerSwimlane
                    key={owner}
                    label={`@${owner}`}
                    lastActive={getLastActiveDate(stacks)}
                    stacks={stacks}
                    selectedBranch={selectedBranch?.name ?? null}
                    onSelectBranch={handleSelectBranch}
                  />
                ))}
              </div>

              {/* Trunk line */}
              <div className="flex items-center gap-2 px-6 pb-2">
                <div className="flex-1 border-t-2 border-dashed border-muted-foreground/30" />
                <span className="text-xs font-mono text-muted-foreground">{repo?.trunk}</span>
                <div className="flex-1 border-t-2 border-dashed border-muted-foreground/30" />
              </div>

              {/* Recently merged history */}
              <RecentlyMerged
                commits={recentlyMerged}
                owner={repo?.owner}
                repo={repo?.repo}
              />
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
              No stacks found
            </div>
          )}
        </div>

        {/* Right: branch detail */}
        <AnimatePresence>
          {selectedBranch && (
            <motion.div
              key="detail-panel"
              className="flex shrink-0"
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 400, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ duration: 0.2, ease: "easeInOut" }}
            >
              <Separator orientation="vertical" />
              <div className="w-[400px] shrink-0 overflow-auto p-4">
                <BranchDetail branch={selectedBranch} />
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
