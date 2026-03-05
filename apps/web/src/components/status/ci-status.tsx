import type { CIResponse } from "@/lib/api";
import { AnimatedCheckmark, AnimatedX, PulsingDot } from "@/components/ui/animated-ci-icons";
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip";

export function CIStatusIcon({ status }: { status?: string }) {
  if (!status || status === "none") return null;

  switch (status) {
    case "passing":
      return <AnimatedCheckmark />;
    case "failing":
      return <AnimatedX />;
    case "pending":
      return <PulsingDot />;
    default:
      return null;
  }
}

export function CheckIcon({ conclusion, status }: { conclusion: string; status: string }) {
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

export function CIStatusWithTooltip({ ci }: { ci?: CIResponse }) {
  if (!ci || ci.status === "none") return null;

  const checks = ci.checks ?? [];
  const passingCount = checks.filter(
    (c) => c.status === "COMPLETED" && c.conclusion === "SUCCESS"
  ).length;
  const total = checks.length;

  const statusColorClass =
    ci.status === "passing"
      ? "text-green-600"
      : ci.status === "failing"
        ? "text-red-600"
        : "text-amber-500";

  if (total === 0) {
    return <CIStatusIcon status={ci.status} />;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`inline-flex items-center gap-1 shrink-0 ${statusColorClass}`}>
          <CIStatusIcon status={ci.status} />
          <span className="text-[10px] font-medium leading-none">
            {passingCount}/{total}
          </span>
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-64">
        <div className="space-y-1 py-0.5">
          {checks.map((check) => (
            <div key={check.name} className="flex items-center gap-2 text-xs">
              <CheckIcon conclusion={check.conclusion} status={check.status} />
              <span className="truncate">{check.name}</span>
            </div>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
