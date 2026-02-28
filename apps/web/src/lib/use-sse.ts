"use client";

import { useEffect, useRef, useCallback } from "react";
import type { FeedEvent } from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type SSECallback = () => void;

/**
 * Hook that connects to the SSE event stream and calls onUpdate
 * whenever stack refresh events are received.
 * Optionally calls onEvent for server-sourced feed events.
 */
export function useSSE(
  onUpdate: SSECallback,
  onEvent?: (event: FeedEvent) => void
) {
  const callbackRef = useRef(onUpdate);
  callbackRef.current = onUpdate;

  const eventCallbackRef = useRef(onEvent);
  eventCallbackRef.current = onEvent;

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

    eventSource.addEventListener("branch_switched", (e) => {
      if (eventCallbackRef.current) {
        try {
          const data = JSON.parse(e.data);
          eventCallbackRef.current({
            kind: "branch_switched",
            timestamp: data.timestamp || new Date().toISOString(),
            branch: data.to,
            detail: `from ${data.from}`,
          });
        } catch {
          // Ignore malformed events
        }
      }
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
