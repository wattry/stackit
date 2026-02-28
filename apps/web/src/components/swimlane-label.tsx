"use client";

interface SwimlaneLabelProps {
  label: string;
  lastActive?: Date;
  color: string;
}

export function SwimlaneLabel({ label, lastActive, color }: SwimlaneLabelProps) {
  return (
    <div
      className="flex items-center justify-center gap-2 px-3 py-1 w-full rounded-b-md"
      style={{ backgroundColor: color }}
    >
      <span className="text-xs font-semibold text-foreground/80">
        {label}
      </span>
      {lastActive && (
        <span className="text-[10px] text-foreground/40">
          active <TimeAgo date={lastActive} />
        </span>
      )}
    </div>
  );
}

/**
 * Generate a soft pastel background color from a string.
 * Returns an HSL color with fixed saturation/lightness and a hue derived from the input.
 */
export function swimlaneColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = ((hash % 360) + 360) % 360;
  return `hsl(${hue} 30% 95%)`;
}

function TimeAgo({ date }: { date: Date }) {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 5) return <>just now</>;
  if (seconds < 60) return <>{seconds}s ago</>;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return <>{minutes}m ago</>;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return <>{hours}h ago</>;
  const days = Math.floor(hours / 24);
  return <>{days}d ago</>;
}
