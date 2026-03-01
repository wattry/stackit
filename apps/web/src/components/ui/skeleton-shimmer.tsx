"use client";

export function Skeleton({ className = "" }: { className?: string }) {
  return (
    <div
      className={`relative overflow-hidden rounded bg-muted ${className}`}
    >
      <div className="absolute inset-0 animate-shimmer bg-gradient-to-r from-transparent via-foreground/5 to-transparent" />
    </div>
  );
}

export function SkeletonCard() {
  return (
    <div className="w-64 shrink-0">
      {/* Header skeleton */}
      <div className="px-1 pb-2 space-y-2">
        <Skeleton className="h-5 w-24" />
        <Skeleton className="h-3 w-32" />
      </div>
      {/* Card stack skeleton */}
      <div className="flex flex-col rounded-lg border bg-card">
        {[0, 1, 2].map((i) => (
          <div
            key={i}
            className="px-3 py-2.5 border-b last:border-b-0 space-y-2"
          >
            <Skeleton className="h-4 w-40" />
            <div className="flex gap-2">
              <Skeleton className="h-3 w-12" />
              <Skeleton className="h-3 w-8" />
              <Skeleton className="h-3 w-16" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function SkeletonSwimlane() {
  return (
    <div className="flex items-center justify-center h-screen">
      <div className="flex flex-col items-end">
        <div className="flex gap-4 items-end p-6 pb-4">
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="animate-fade-in-up"
              style={{ animationDelay: `${i * 100}ms` }}
            >
              <SkeletonCard />
            </div>
          ))}
        </div>
        {/* Trunk line skeleton */}
        <div className="flex items-center gap-2 px-6 pb-6 w-full">
          <Skeleton className="flex-1 h-0.5" />
          <Skeleton className="h-3 w-10" />
          <Skeleton className="flex-1 h-0.5" />
        </div>
      </div>
    </div>
  );
}
