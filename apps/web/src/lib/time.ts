export function formatTimeAgo(input: Date | string): string {
  const date = typeof input === "string" ? new Date(input) : input;
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

  if (seconds < 5) return "just now";
  if (seconds < 60) return `${seconds}s ago`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export type TimeGroup = "Today" | "Yesterday" | "This week" | "Earlier";

export function timeGroup(dateStr: string): TimeGroup {
  const now = new Date();
  const date = new Date(dateStr);
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const startOfYesterday = new Date(startOfToday.getTime() - 86_400_000);
  const startOfWeek = new Date(startOfToday.getTime() - startOfToday.getDay() * 86_400_000);

  if (date >= startOfToday) return "Today";
  if (date >= startOfYesterday) return "Yesterday";
  if (date >= startOfWeek) return "This week";
  return "Earlier";
}

export function groupByTime<T extends { date: string }>(
  items: T[]
): { label: TimeGroup; items: T[] }[] {
  const order: TimeGroup[] = ["Today", "Yesterday", "This week", "Earlier"];
  const map = new Map<TimeGroup, T[]>();
  for (const item of items) {
    const g = timeGroup(item.date);
    const list = map.get(g);
    if (list) {
      list.push(item);
    } else {
      map.set(g, [item]);
    }
  }
  return order.filter((l) => map.has(l)).map((l) => ({ label: l, items: map.get(l)! }));
}
