/** Status config for stack status footer buttons (used in swimlane columns). */
export interface StackStatusStyle {
  label: string;
  bg: string;
  selectedBg: string;
  text: string;
  shadow: string;
}

export const stackStatusStyles: Record<string, StackStatusStyle> = {
  shippable: {
    label: "Ready to ship",
    bg: "bg-green-100 dark:bg-green-950",
    selectedBg: "bg-green-200 dark:bg-green-900",
    text: "text-green-700 dark:text-green-400",
    shadow: "shadow-[0_2px_8px_rgba(34,197,94,0.2)] dark:shadow-[0_2px_8px_rgba(34,197,94,0.15)]",
  },
  pending: {
    label: "Needs restack",
    bg: "bg-amber-100 dark:bg-amber-950",
    selectedBg: "bg-amber-200 dark:bg-amber-900",
    text: "text-amber-700 dark:text-amber-400",
    shadow: "shadow-[0_2px_8px_rgba(245,158,11,0.2)] dark:shadow-[0_2px_8px_rgba(245,158,11,0.15)]",
  },
  blocked: {
    label: "Blocked",
    bg: "bg-red-100 dark:bg-red-950",
    selectedBg: "bg-red-200 dark:bg-red-900",
    text: "text-red-700 dark:text-red-400",
    shadow: "shadow-[0_2px_8px_rgba(239,68,68,0.2)] dark:shadow-[0_2px_8px_rgba(239,68,68,0.15)]",
  },
  incomplete: {
    label: "Incomplete",
    bg: "bg-muted",
    selectedBg: "bg-muted",
    text: "text-muted-foreground",
    shadow: "shadow-[0_2px_8px_rgba(0,0,0,0.05)] dark:shadow-[0_2px_8px_rgba(0,0,0,0.15)]",
  },
};

/** Status config for stack detail panel (label, description, color). */
export interface StackStatusInfo {
  label: string;
  description: string;
  color: string;
}

export const stackStatusInfo: Record<string, StackStatusInfo> = {
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
