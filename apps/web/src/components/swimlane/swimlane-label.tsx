"use client";

interface SwimlaneLabelProps {
  label: string;
  lastActive?: Date;
  color: string;
  compact?: boolean;
}

export function SwimlaneLabel({
  label,
  lastActive,
  color,
  compact = false,
}: SwimlaneLabelProps) {
  return (
    <div
      className={`flex items-center justify-center w-full rounded-b-xl ${compact ? "gap-1.5 px-2 py-0.5" : "gap-2 px-3 py-1"}`}
      style={{ backgroundColor: color }}
    >
      <span className={`${compact ? "text-[11px]" : "text-xs"} font-semibold text-foreground/80`}>
        {label}
      </span>
      {lastActive && (
        <span className={`${compact ? "text-[9px]" : "text-[10px]"} text-foreground/60`}>
          active {formatLastActive(lastActive)}
        </span>
      )}
    </div>
  );
}

function swimlaneHue(name: string): number {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return ((hash % 360) + 360) % 360;
}

/**
 * Generate a soft pastel background color from a string.
 * Returns a CSS `light-dark()` value that adapts to the color scheme.
 */
export function swimlaneColor(name: string): string {
  const hue = swimlaneHue(name);
  return `light-dark(hsl(${hue} 30% 95%), hsl(${hue} 20% 18%))`;
}

/** Saturated accent color for borders/highlights derived from the swimlane hue. */
export function swimlaneAccent(name: string): string {
  const hue = swimlaneHue(name);
  return `light-dark(hsl(${hue} 50% 65%), hsl(${hue} 40% 45%))`;
}

function formatLastActive(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(date);
}
