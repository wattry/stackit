"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import {
  fetchView,
  type RepoResponse,
  type StackDetail,
} from "@/lib/api";
import { useSSE } from "@/lib/use-sse";

interface RepoState {
  repo: RepoResponse | null;
  stackDetails: StackDetail[];
  loading: boolean;
  error: string | null;
  lastUpdated: Date | null;
  refresh: () => void;
}

const RepoContext = createContext<RepoState | null>(null);

export function useRepo() {
  const ctx = useContext(RepoContext);
  if (!ctx) throw new Error("useRepo must be used within RepoProvider");
  return ctx;
}

export function RepoProvider({ children }: { children: ReactNode }) {
  const [repo, setRepo] = useState<RepoResponse | null>(null);
  const [stackDetails, setStackDetails] = useState<StackDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const loadData = useCallback(async () => {
    try {
      const view = await fetchView();
      setRepo(view.repo);

      // TODO: Remove sample stacks — for UI development only
      const sampleStacks: StackDetail[] = [
        {
          rootBranch: "sample/auth-refactor",
          title: "Refactor auth middleware",
          status: "shippable",
          branchCount: 2,
          prCount: 2,
          isCurrent: false,
          owner: "teammate-alice",
          branches: [
            {
              name: "sample/auth-refactor",
              depth: 0,
              isCurrent: false,
              needsRestack: false,
              isLocked: false,
              isFrozen: false,
              revision: "abc1234",
              commitDate: new Date().toISOString(),
              commitAuthor: "teammate-alice",
              commitCount: 3,
              linesAdded: 120,
              linesDeleted: 45,
              pr: {
                number: 101,
                title: "Refactor auth middleware",
                state: "OPEN",
                url: "#",
                isDraft: false,
                base: "main",
              },
            },
            {
              name: "sample/auth-tests",
              parent: "sample/auth-refactor",
              depth: 1,
              isCurrent: false,
              needsRestack: false,
              isLocked: false,
              isFrozen: false,
              revision: "def5678",
              commitDate: new Date().toISOString(),
              commitAuthor: "teammate-alice",
              commitCount: 1,
              linesAdded: 80,
              linesDeleted: 0,
              pr: {
                number: 102,
                title: "Add auth middleware tests",
                state: "OPEN",
                url: "#",
                isDraft: false,
                base: "sample/auth-refactor",
              },
            },
          ],
        },
        {
          rootBranch: "sample/fix-pagination",
          title: "Fix pagination off-by-one",
          status: "pending",
          branchCount: 1,
          prCount: 1,
          isCurrent: false,
          owner: "teammate-bob",
          branches: [
            {
              name: "sample/fix-pagination",
              depth: 0,
              isCurrent: false,
              needsRestack: true,
              isLocked: false,
              isFrozen: false,
              revision: "fed9876",
              commitDate: new Date().toISOString(),
              commitAuthor: "teammate-bob",
              commitCount: 1,
              linesAdded: 5,
              linesDeleted: 3,
              pr: {
                number: 200,
                title: "Fix pagination off-by-one",
                state: "OPEN",
                url: "#",
                isDraft: true,
                base: "main",
              },
            },
          ],
        },
      ];

      setStackDetails([...view.stacks, ...sampleStacks]);

      setError(null);
      setLastUpdated(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    loadData();
  }, [loadData]);

  // SSE updates trigger refresh
  useSSE(loadData);

  return (
    <RepoContext.Provider
      value={{
        repo,
        stackDetails,
        loading,
        error,
        lastUpdated,
        refresh: loadData,
      }}
    >
      {children}
    </RepoContext.Provider>
  );
}
