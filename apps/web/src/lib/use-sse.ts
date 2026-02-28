"use client";

import { useEffect, useRef, useCallback } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type SSECallback = () => void;

/**
 * Hook that connects to the SSE event stream and calls onUpdate
 * whenever stack refresh events are received.
 */
export function useSSE(onUpdate: SSECallback) {
  const callbackRef = useRef(onUpdate);
  callbackRef.current = onUpdate;

  const stableCallback = useCallback(() => {
    callbackRef.current();
  }, []);

  useEffect(() => {
    const eventSource = new EventSource(`${API_BASE}/api/events`);

    eventSource.addEventListener("stacks_updated", () => {
      stableCallback();
    });

    eventSource.addEventListener("branch_changed", () => {
      stableCallback();
    });

    eventSource.addEventListener("refresh", () => {
      stableCallback();
    });

    eventSource.onerror = () => {
      // EventSource will auto-reconnect
    };

    return () => {
      eventSource.close();
    };
  }, [stableCallback]);
}
