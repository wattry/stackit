"use client";

import { useState } from "react";
import { motion } from "motion/react";
import type { StackDetail as StackDetailType, BranchResponse } from "@/lib/api";
import { submitStack } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  PRBadge,
  CIStatusBadge,
  DiffStats,
} from "@/components/status/status-badge";
import { useRepo } from "@/components/providers/repo-provider";

interface StackDetailPanelProps {
  stack: StackDetailType;
}

const statusConfig: Record<string, { label: string; description: string; color: string }> = {
  shippable: {
    label: "Ready to ship",
    description: "All branches have PRs, none need restack, and none are locked.",
    color: "text-green-700 dark:text-green-400",
  },
  pending: {
    label: "Needs restack",
    description: "One or more branches need to be restacked because a parent branch has changed.",
    color: "text-amber-700 dark:text-amber-400",
  },
  blocked: {
    label: "Blocked",
    description: "One or more branches are locked.",
    color: "text-red-700 dark:text-red-400",
  },
  incomplete: {
    label: "Incomplete",
    description: "One or more branches are missing a pull request.",
    color: "text-muted-foreground",
  },
};

export function StackDetailPanel({ stack }: StackDetailPanelProps) {
  const status = statusConfig[stack.status] || statusConfig.incomplete;
  const issues = collectIssues(stack.branches);
  const { refresh } = useRepo();

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<{ success: boolean; message: string } | null>(null);

  const handleSubmit = async () => {
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const result = await submitStack(stack.rootBranch);
      setSubmitResult({ success: result.success, message: result.message });
      refresh();
    } catch (err) {
      setSubmitResult({
        success: false,
        message: err instanceof Error ? err.message : "Submit failed",
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <motion.div
      key={`stack-${stack.rootBranch}`}
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
    >
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Stack overview</CardTitle>
            <span className={`text-xs font-medium ${status.color}`}>
              {status.label}
            </span>
          </div>
          {stack.title && (
            <p className="text-sm text-muted-foreground">{stack.title}</p>
          )}
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Status explanation */}
          <p className="text-sm text-muted-foreground">{status.description}</p>

          {/* Issues */}
          {issues.length > 0 && (
            <>
              <Separator />
              <div className="space-y-1.5">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Issues
                </h4>
                <div className="flex flex-col gap-1">
                  {issues.map((issue) => (
                    <div
                      key={issue.key}
                      className="flex items-center gap-2 text-sm rounded-md bg-amber-50 dark:bg-amber-950 px-2 py-1 ring-1 ring-inset ring-amber-600/20"
                    >
                      <span className="text-amber-700 dark:text-amber-300">
                        {issue.message}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}

          <Separator />

          {/* All branches (leaf → root) */}
          <div className="space-y-1.5">
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Branches ({stack.branches.length})
            </h4>
            <div className="flex flex-col gap-2">
              {[...stack.branches].reverse().map((branch) => (
                <BranchRow key={branch.name} branch={branch} />
              ))}
            </div>
          </div>

          <Separator />

          {/* Submit button */}
          <button
            onClick={handleSubmit}
            disabled={submitting || stack.status === "blocked"}
            className="w-full rounded-md px-3 py-2 text-sm font-medium transition-colors bg-green-600 text-white hover:bg-green-700 dark:bg-green-700 dark:hover:bg-green-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {submitting ? "Submitting..." : "Submit stack"}
          </button>

          {/* Submit result */}
          {submitResult && (
            <div
              className={`rounded-md px-3 py-2 text-sm ${
                submitResult.success
                  ? "bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 ring-1 ring-inset ring-green-600/20"
                  : "bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 ring-1 ring-inset ring-red-600/20"
              }`}
            >
              {submitResult.message}
            </div>
          )}
        </CardContent>
      </Card>
    </motion.div>
  );
}

function BranchRow({ branch }: { branch: BranchResponse }) {
  return (
    <div className="flex flex-col gap-1 rounded-md border px-3 py-2">
      <div className="flex items-center justify-between gap-2 min-w-0">
        <span className="text-sm font-mono truncate">{branch.name}</span>
        <div className="flex items-center gap-1.5 shrink-0">
          {branch.pr ? (
            <PRBadge pr={branch.pr} />
          ) : (
            <span className="text-xs text-muted-foreground">no PR</span>
          )}
          <CIStatusBadge ci={branch.ci} />
        </div>
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
        <DiffStats added={branch.linesAdded} deleted={branch.linesDeleted} />
        {branch.needsRestack && (
          <span className="text-amber-600 dark:text-amber-400">needs restack</span>
        )}
        {branch.isLocked && (
          <span className="text-red-600 dark:text-red-400">
            locked{branch.lockReason ? ` (${branch.lockReason})` : ""}
          </span>
        )}
      </div>
    </div>
  );
}

interface Issue {
  key: string;
  message: string;
}

function collectIssues(branches: BranchResponse[]): Issue[] {
  const issues: Issue[] = [];

  for (const b of branches) {
    if (!b.pr) {
      issues.push({ key: `no-pr-${b.name}`, message: `${b.name} has no PR` });
    }
    if (b.needsRestack) {
      issues.push({ key: `restack-${b.name}`, message: `${b.name} needs restack` });
    }
    if (b.isLocked) {
      issues.push({
        key: `locked-${b.name}`,
        message: `${b.name} is locked${b.lockReason ? ` (${b.lockReason})` : ""}`,
      });
    }
  }

  return issues;
}
