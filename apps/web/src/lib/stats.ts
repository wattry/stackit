import type { BranchResponse } from "@/lib/api";

export interface StackStats {
  prCount: number;
  approvedCount: number;
  ciPassingCount: number;
  totalAdded: number;
  totalDeleted: number;
  total: number;
}

export function computeStackStats(branches: BranchResponse[]): StackStats {
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
