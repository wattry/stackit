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

export function getLastActiveDate(stacks: StackDetail[]): Date | undefined {
  let latest = 0;
  for (const stack of stacks) {
    for (const branch of stack.branches) {
      const t = new Date(branch.commitDate).getTime();
      if (t > latest) latest = t;
    }
  }
  return latest ? new Date(latest) : undefined;
}
