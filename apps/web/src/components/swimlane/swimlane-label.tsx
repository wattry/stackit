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
      className={`flex items-center justify-center w-full rounded-b-md ${compact ? "gap-1.5 px-2 py-0.5" : "gap-2 px-3 py-1"}`}
      style={{ backgroundColor: color }}
    >
      <span className={`${compact ? "text-[11px]" : "text-xs"} font-semibold text-foreground/80`}>
        {label}
      </span>
      {lastActive && (
        <span className={`${compact ? "text-[9px]" : "text-[10px]"} text-foreground/40`}>
          active {formatLastActive(lastActive)}
        </span>
      )}
    </div>
  );
}

/**
 * Generate a soft pastel background color from a string.
 * Returns a CSS `light-dark()` value that adapts to the color scheme.
 * Light mode uses high lightness (95%), dark mode uses low lightness (18%).
 */
export function swimlaneColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = ((hash % 360) + 360) % 360;
  return `light-dark(hsl(${hue} 30% 95%), hsl(${hue} 20% 18%))`;
}

function formatLastActive(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(date);
}
