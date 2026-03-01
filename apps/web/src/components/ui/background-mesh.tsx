"use client";

export function BackgroundMesh() {
  return (
    <div
      className="fixed inset-0 -z-10 overflow-hidden pointer-events-none"
      aria-hidden="true"
    >
      <div
        className="absolute -top-1/4 -left-1/4 w-[50vw] h-[50vw] rounded-full opacity-20 blur-[120px]"
        style={{
          background: "var(--gradient-start)",
          animation: "mesh-float 20s ease-in-out infinite",
        }}
      />
      <div
        className="absolute top-1/3 -right-1/4 w-[40vw] h-[40vw] rounded-full opacity-15 blur-[120px]"
        style={{
          background: "var(--gradient-mid)",
          animation: "mesh-float 25s ease-in-out infinite 5s",
        }}
      />
      <div
        className="absolute -bottom-1/4 left-1/3 w-[45vw] h-[45vw] rounded-full opacity-10 blur-[120px]"
        style={{
          background: "var(--gradient-end)",
          animation: "mesh-float 22s ease-in-out infinite 10s",
        }}
      />
    </div>
  );
}
