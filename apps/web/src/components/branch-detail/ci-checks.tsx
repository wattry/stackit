"use client";

import type { CIResponse } from "@/lib/api";
import { AnimatedCheckmark, AnimatedX, PulsingDot } from "@/components/ui/animated-ci-icons";

export function CIChecks({ ci }: { ci?: CIResponse }) {
  if (!ci || !ci.checks || ci.checks.length === 0) return null;

  return (
    <div className="space-y-1">
      <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
        CI Checks
      </h4>
      <div className="space-y-0.5">
        {ci.checks.map((check, i) => (
          <div
            key={check.name}
            className="flex items-center gap-2 text-sm"
          >
            <CheckIcon conclusion={check.conclusion} status={check.status} />
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

function CheckIcon({ conclusion, status }: { conclusion: string; status: string }) {
  if (status !== "COMPLETED") {
    return <PulsingDot />;
  }
  switch (conclusion) {
    case "SUCCESS":
      return <AnimatedCheckmark />;
    case "FAILURE":
      return <AnimatedX />;
    case "NEUTRAL":
      return <span className="inline-block w-2.5 h-2.5 rounded-full bg-muted-foreground/40" />;
    default:
      return <PulsingDot />;
  }
}
