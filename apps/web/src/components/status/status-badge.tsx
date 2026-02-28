"use client";

import { Github } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { CIResponse, PRResponse } from "@/lib/api";

export function CIStatusBadge({ ci }: { ci?: CIResponse }) {
  if (!ci || ci.status === "none") return null;

  const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
    passing: "default",
    pending: "secondary",
    failing: "destructive",
  };

  const labels: Record<string, string> = {
    passing: "CI passing",
    pending: "CI pending",
    failing: "CI failing",
  };

  return (
    <Badge variant={variants[ci.status] || "outline"} className="text-xs">
      {labels[ci.status] || ci.status}
    </Badge>
  );
}

export function ReviewBadge({ ci }: { ci?: CIResponse }) {
  if (!ci || !ci.reviewDecision) return null;

  const config: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
    APPROVED: { label: "Approved", variant: "default" },
    CHANGES_REQUESTED: { label: "Changes requested", variant: "destructive" },
    REVIEW_REQUIRED: { label: "Review required", variant: "secondary" },
  };

  const c = config[ci.reviewDecision];
  if (!c) return null;

  return (
    <Badge variant={c.variant} className="text-xs">
      {c.label}
    </Badge>
  );
}

export function PRBadge({ pr }: { pr?: PRResponse }) {
  if (!pr) return null;

  const stateColors: Record<string, string> = {
    OPEN: "text-green-600",
    MERGED: "text-purple-600",
    CLOSED: "text-red-600",
  };

  return (
    <a
      href={pr.url}
      target="_blank"
      rel="noopener noreferrer"
      className={`inline-flex items-center gap-1 text-xs font-mono hover:underline ${stateColors[pr.state] || ""}`}
    >
      <Github className="w-3 h-3" />
      #{pr.number}
      {pr.isDraft && " (draft)"}
    </a>
  );
}

export function DiffStats({
  added,
  deleted,
}: {
  added: number;
  deleted: number;
}) {
  if (added === 0 && deleted === 0) return null;

  return (
    <span className="text-xs font-mono">
      {added > 0 && <span className="text-green-600">+{added}</span>}
      {added > 0 && deleted > 0 && " / "}
      {deleted > 0 && <span className="text-red-600">-{deleted}</span>}
    </span>
  );
}

export function StackStatusBadge({ status }: { status: string }) {
  const config: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
    shippable: { label: "Ready to ship", variant: "default" },
    pending: { label: "Needs restack", variant: "secondary" },
    blocked: { label: "Blocked", variant: "destructive" },
    incomplete: { label: "Incomplete", variant: "outline" },
  };

  const c = config[status] || { label: status, variant: "outline" as const };

  return (
    <Badge variant={c.variant} className="text-xs">
      {c.label}
    </Badge>
  );
}
