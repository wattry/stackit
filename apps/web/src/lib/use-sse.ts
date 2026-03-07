"use client";

import { useEffect, useEffectEvent } from "react";
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
  const handleUpdate = useEffectEvent(onUpdate);
  const handleEvent = useEffectEvent((event: FeedEvent) => {
    onEvent?.(event);
  });

  useEffect(() => {
    const eventSource = new EventSource(`${API_BASE}/api/events`);

    eventSource.addEventListener("stacks_updated", () => {
      handleUpdate();
    });

    eventSource.addEventListener("branch_changed", () => {
      handleUpdate();
    });

    eventSource.addEventListener("refresh", () => {
      handleUpdate();
    });

    eventSource.addEventListener("branch_switched", (e) => {
      try {
        const data = JSON.parse(e.data);
        handleEvent({
          kind: "branch_switched",
          timestamp: data.timestamp || new Date().toISOString(),
          branch: data.to,
          detail: `from ${data.from}`,
        });
      } catch {
        // Ignore malformed events
      }
      handleUpdate();
    });

    eventSource.onerror = () => {
      // EventSource will auto-reconnect
    };

    return () => {
      eventSource.close();
    };
  }, []);
}
