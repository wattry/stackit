"use client";

import type { BranchResponse } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  PRBadge,
  CIStatusBadge,
  ReviewBadge,
  DiffStats,
} from "@/components/status/status-badge";
import { CommitList } from "./commit-list";
import { CIChecks } from "./ci-checks";

interface BranchDetailProps {
  branch: BranchResponse;
}

export function BranchDetail({ branch }: BranchDetailProps) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base font-mono">{branch.name}</CardTitle>
          <div className="flex items-center gap-2">
            <PRBadge pr={branch.pr} />
            <CIStatusBadge ci={branch.ci} />
            <ReviewBadge ci={branch.ci} />
          </div>
        </div>
        {branch.pr && (
          <p className="text-sm text-muted-foreground">{branch.pr.title}</p>
        )}
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Metadata row */}
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
          {branch.parent && <span>Parent: {branch.parent}</span>}
          <span>
            {branch.commitCount} commit{branch.commitCount !== 1 && "s"}
          </span>
          <DiffStats added={branch.linesAdded} deleted={branch.linesDeleted} />
          {branch.commitAuthor && <span>{branch.commitAuthor}</span>}
        </div>

        {/* Warnings */}
        <Warnings branch={branch} />

        <Separator />

        {/* Commits */}
        <CommitList commits={branch.commits} />

        {/* CI Checks */}
        <CIChecks ci={branch.ci} />
      </CardContent>
    </Card>
  );
}

function Warnings({ branch }: { branch: BranchResponse }) {
  const warnings: string[] = [];

  if (branch.needsRestack) warnings.push("Needs restack");
  if (branch.isLocked) warnings.push(`Locked (${branch.lockReason || "unknown"})`);
  if (branch.isFrozen) warnings.push("Frozen");
  if (branch.remoteStatus?.missingRemote) warnings.push("Not yet pushed");
  if (branch.remoteStatus?.behind) warnings.push("Behind remote");
  if (branch.remoteStatus?.diverged) warnings.push("Diverged from remote");
  if (!branch.pr) warnings.push("No PR");

  if (warnings.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1">
      {warnings.map((w) => (
        <span
          key={w}
          className="inline-flex items-center rounded-md bg-amber-50 dark:bg-amber-950 px-2 py-0.5 text-xs text-amber-700 dark:text-amber-300 ring-1 ring-inset ring-amber-600/20"
        >
          {w}
        </span>
      ))}
    </div>
  );
}
