export function DiffSkeleton() {
  return (
    <div className="grid gap-3 lg:grid-cols-[18rem,minmax(0,1fr)] lg:items-start lg:gap-4">
      {/* File explorer skeleton */}
      <div className="rounded-md border bg-card/60">
        <div className="px-3 py-2 text-xs font-medium text-muted-foreground">
          Files
        </div>
        <div className="p-2 space-y-1.5">
          {[0, 1, 2, 3, 4].map((i) => (
            <div key={i} className="flex items-center gap-2 px-1.5 py-1">
              <div className="h-3 w-3 shrink-0 rounded bg-muted animate-pulse" />
              <div
                className="h-3 rounded bg-muted animate-pulse"
                style={{ width: `${60 + (i * 17) % 30}%` }}
              />
            </div>
          ))}
        </div>
      </div>
      {/* Diff pane skeleton */}
      <div className="space-y-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="rounded-md border bg-card/60 overflow-hidden">
            <div className="flex items-center gap-2 border-b px-3 py-2">
              <div className="h-3 w-3 rounded bg-muted animate-pulse" />
              <div
                className="h-3 rounded bg-muted animate-pulse"
                style={{ width: `${100 + (i * 37) % 80}px` }}
              />
            </div>
            <div className="p-3 space-y-2">
              {[0, 1, 2, 3].map((j) => (
                <div
                  key={j}
                  className="h-3 rounded bg-muted/60 animate-pulse"
                  style={{ width: `${40 + ((i + j) * 23) % 55}%` }}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
