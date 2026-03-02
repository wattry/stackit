import { GitBranch } from "lucide-react";

export function DetailEmptyState() {
  return (
    <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground px-6">
      <GitBranch className="w-8 h-8 opacity-40" />
      <p className="text-sm text-center">
        Select a branch or stack to view details
      </p>
    </div>
  );
}
