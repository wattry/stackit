"use client";

import type { CIResponse } from "@/lib/api";

export function CIChecks({ ci }: { ci?: CIResponse }) {
  if (!ci || !ci.checks || ci.checks.length === 0) return null;

  return (
    <div className="space-y-1">
      <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
        CI Checks
      </h4>
      <div className="space-y-0.5">
        {ci.checks.map((check) => (
          <div key={check.name} className="flex items-center gap-2 text-sm">
            <span>{getCheckIcon(check.conclusion)}</span>
            <span className="truncate">{check.name}</span>
            <span className="text-xs text-muted-foreground ml-auto">
              {check.status === "COMPLETED" ? check.conclusion : check.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function getCheckIcon(conclusion: string): string {
  switch (conclusion) {
    case "SUCCESS":
      return "\u2705";
    case "FAILURE":
      return "\u274C";
    case "NEUTRAL":
      return "\u26AA";
    default:
      return "\u23F3";
  }
}
