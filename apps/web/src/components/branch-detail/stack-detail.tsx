"use client";

import { useState } from "react";
import { motion } from "motion/react";
import type { StackDetail as StackDetailType, BranchResponse, SubmitResponse } from "@/lib/api";
import { submitStack } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  PRBadge,
  CIStatusBadge,
  DiffStats,
} from "@/components/status/status-badge";
import { StackDescription } from "@/components/stack-column";
import { useRepo } from "@/components/providers/repo-provider";
import { cn } from "@/lib/utils";

interface StackDetailPanelProps {
  stack: StackDetailType;
  onSelectBranch?: (branch: BranchResponse) => void;
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

function computeStackStats(branches: BranchResponse[]) {
  let prCount = 0;
  let approvedCount = 0;
  let ciPassingCount = 0;
  let totalAdded = 0;
  let totalDeleted = 0;

  for (const b of branches) {
    if (b.pr) prCount++;
    if (b.ci?.reviewDecision === "APPROVED") approvedCount++;
    if (b.ci?.status === "passing") ciPassingCount++;
    totalAdded += b.linesAdded;
    totalDeleted += b.linesDeleted;
  }

  return { prCount, approvedCount, ciPassingCount, totalAdded, totalDeleted, total: branches.length };
}

export function StackDetailPanel({ stack, onSelectBranch }: StackDetailPanelProps) {
  const status = statusConfig[stack.status] || statusConfig.incomplete;
  const issues = collectIssues(stack.branches);
  const stats = computeStackStats(stack.branches);
  const { refresh } = useRepo();

  const allHavePRs = stack.branches.every((b) => b.pr != null);

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<SubmitResponse | null>(null);

  const handleSubmit = async () => {
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const result = await submitStack(stack.rootBranch);
      setSubmitResult(result);
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
      <Card className="overflow-hidden">
        <CardHeader className="pb-3 min-w-0">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Stack overview</CardTitle>
            <span className={`text-xs font-medium ${status.color}`}>
              {status.label}
            </span>
          </div>
          {stack.title && (
            <p className="text-sm text-muted-foreground">{stack.title}</p>
          )}
          {stack.description && (
            <StackDescription text={stack.description} />
          )}
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Status explanation */}
          <p className="text-sm text-muted-foreground">{status.description}</p>

          {/* Aggregate stats */}
          <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <span>{stats.prCount}/{stats.total} PRs</span>
            <span>{stats.approvedCount}/{stats.total} approved</span>
            <span>{stats.ciPassingCount}/{stats.total} CI passing</span>
            <DiffStats added={stats.totalAdded} deleted={stats.totalDeleted} />
          </div>

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
                <BranchRow
                  key={branch.name}
                  branch={branch}
                  onClick={onSelectBranch ? () => onSelectBranch(branch) : undefined}
                />
              ))}
            </div>
          </div>

          <Separator />

          {/* Submit button — hidden when all branches already have PRs */}
          {!allHavePRs && (
            <button
              onClick={handleSubmit}
              disabled={submitting || stack.status === "blocked"}
              className="w-full rounded-md px-3 py-2 text-sm font-medium transition-colors bg-green-600 text-white hover:bg-green-700 dark:bg-green-700 dark:hover:bg-green-600 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "Submitting..." : "Submit stack"}
            </button>
          )}

          {/* Submit result */}
          {submitResult && (
            <div className="space-y-2">
              <div
                className={`rounded-md px-3 py-2 text-sm ${
                  submitResult.success
                    ? "bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 ring-1 ring-inset ring-green-600/20"
                    : "bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 ring-1 ring-inset ring-red-600/20"
                }`}
              >
                {submitResult.message}
              </div>
              {submitResult.branches && submitResult.branches.length > 0 && (
                <div className="flex flex-col gap-1">
                  {submitResult.branches.map((b) => (
                    <div
                      key={b.name}
                      className="flex items-center justify-between text-xs px-2 py-1 rounded-md bg-muted"
                    >
                      <span className="font-mono truncate">{b.name}</span>
                      <span className={cn(
                        "shrink-0 ml-2",
                        b.error ? "text-red-600" : "text-green-600"
                      )}>
                        {b.error || b.status}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </motion.div>
  );
}

function BranchRow({ branch, onClick }: { branch: BranchResponse; onClick?: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex flex-col gap-1 rounded-md border px-3 py-2 text-left transition-colors w-full",
        onClick && "cursor-pointer hover:bg-muted/50",
        branch.isCurrent && "border-l-2 border-l-green-500"
      )}
    >
      <div className="flex items-center justify-between gap-2 min-w-0">
        <span className="flex items-center gap-1.5 text-sm font-mono truncate min-w-0" title={branch.name}>
          {branch.isCurrent && (
            <span className="inline-block w-2 h-2 rounded-full bg-green-500 shrink-0" />
          )}
          {branch.name}
        </span>
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
    </button>
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
