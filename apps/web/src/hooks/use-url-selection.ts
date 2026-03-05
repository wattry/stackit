"use client";

import { useCallback, useMemo, useState } from "react";
import type { BranchResponse, StackDetail } from "@/lib/api";

export type Selection =
  | { type: "branch"; name: string }
  | { type: "stack"; rootBranch: string };

function readInitialSelection(): Selection | null {
  if (typeof window === "undefined") return null;
  const params = new URLSearchParams(window.location.search);
  const branch = params.get("branch");
  if (branch) return { type: "branch", name: branch };
  const stack = params.get("stack");
  if (stack) return { type: "stack", rootBranch: stack };
  return null;
}

function updateUrl(params: Record<string, string | null>) {
  const url = new URL(window.location.href);
  for (const [key, value] of Object.entries(params)) {
    if (value === null) {
      url.searchParams.delete(key);
    } else {
      url.searchParams.set(key, value);
    }
  }
  window.history.replaceState({}, "", url.toString());
}

export function useUrlSelection(stackDetails: StackDetail[]) {
  const [selection, setSelection] = useState<Selection | null>(readInitialSelection);

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

  const selectedBranchStack = useMemo(() => {
    if (!selectedBranch) return null;
    return (
      stackDetails.find((s) =>
        s.branches.some((b) => b.name === selectedBranch.name)
      ) ?? null
    );
  }, [selectedBranch, stackDetails]);

  const handleSelectBranch = useCallback((branch: BranchResponse | null) => {
    if (branch) {
      setSelection({ type: "branch", name: branch.name });
      updateUrl({ branch: branch.name, stack: null });
    } else {
      setSelection(null);
      updateUrl({ branch: null, stack: null });
    }
  }, []);

  const handleClearSelection = useCallback(() => {
    setSelection(null);
    updateUrl({ branch: null, stack: null });
  }, []);

  const handleSelectStack = useCallback((stack: StackDetail) => {
    setSelection((prev) => {
      const deselecting = prev?.type === "stack" && prev.rootBranch === stack.rootBranch;
      const next = deselecting ? null : { type: "stack" as const, rootBranch: stack.rootBranch };

      queueMicrotask(() => {
        if (next) {
          updateUrl({ branch: null, stack: stack.rootBranch });
        } else {
          updateUrl({ branch: null, stack: null });
        }
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

  return {
    selection,
    selectedBranch,
    selectedStack,
    selectedBranchStack,
    handleSelectBranch,
    handleClearSelection,
    handleSelectStack,
    handleNavigateToBranch,
    handleStackBranchSelect,
  };
}
