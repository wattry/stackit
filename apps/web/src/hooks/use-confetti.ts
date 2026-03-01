"use client";

import { useCallback } from "react";
import confetti from "canvas-confetti";

export function useConfetti() {
  return useCallback(() => {
    confetti({
      particleCount: 80,
      spread: 70,
      origin: { y: 0.6 },
      disableForReducedMotion: true,
    });
  }, []);
}
