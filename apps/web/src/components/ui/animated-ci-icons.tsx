"use client";

export function AnimatedCheckmark({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className={`w-4 h-4 ${className}`}
      aria-label="CI passing"
    >
      <path
        d="M3 8.5 L6.5 12 L13 4"
        fill="none"
        className="stroke-green-600"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeDasharray={24}
        strokeDashoffset={0}
        style={{ animation: "checkmark-draw 0.4s ease-out" }}
      />
    </svg>
  );
}

export function AnimatedX({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className={`w-4 h-4 animate-shake ${className}`}
      aria-label="CI failing"
    >
      <path
        d="M4 4 L12 12 M12 4 L4 12"
        fill="none"
        className="stroke-red-600"
        strokeWidth={2}
        strokeLinecap="round"
      />
    </svg>
  );
}

export function PulsingDot({ className = "" }: { className?: string }) {
  return (
    <span
      className={`inline-block w-2.5 h-2.5 rounded-full bg-amber-500 animate-pulse-dot ${className}`}
      aria-label="CI pending"
    />
  );
}
