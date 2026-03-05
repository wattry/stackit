import type { StackDetail } from "@/lib/api";

export function groupStacksByOwner(
  stackDetails: StackDetail[],
  currentUser: string | undefined
): {
  yourStacks: StackDetail[];
  otherOwners: [string, StackDetail[]][];
} {
  const yours: StackDetail[] = [];
  const others = new Map<string, StackDetail[]>();
  for (const stack of stackDetails) {
    if (!stack.owner || stack.owner === currentUser) {
      yours.push(stack);
    } else {
      const existing = others.get(stack.owner);
      if (existing) {
        existing.push(stack);
      } else {
        others.set(stack.owner, [stack]);
      }
    }
  }
  const sortedOthers = [...others.entries()].sort(([a], [b]) =>
    a.localeCompare(b)
  );
  return { yourStacks: yours, otherOwners: sortedOthers };
}
