import { useState } from "react";

import { apiDelete, apiPut } from "@/shared/api/client";
import { useSession } from "@/features/auth/api";
import type { Post, ReactionState } from "@/shared/types";

// The reaction kinds and their glyphs, in display order.
const KINDS: { kind: string; glyph: string; label: string }[] = [
  { kind: "like", glyph: "♥", label: "Like" },
  { kind: "unicorn", glyph: "🦄", label: "Unicorn" },
  { kind: "mind_blown", glyph: "🤯", label: "Mind blown" },
];

interface ReactionBarProps {
  post: Post;
}

/** Reaction and bookmark controls for a post. Both update optimistically and
 * roll back if the server rejects the change. */
export function ReactionBar({ post }: ReactionBarProps) {
  const { user } = useSession();
  const [count, setCount] = useState(post.stats.reactions);
  const [reacted, setReacted] = useState<Set<string>>(
    () => new Set(post.viewer_state?.reacted ?? []),
  );
  const [bookmarked, setBookmarked] = useState(post.viewer_state?.bookmarked ?? false);
  const [busy, setBusy] = useState(false);

  async function toggleReaction(kind: string) {
    if (!user || busy) return;
    const had = reacted.has(kind);

    // Optimistic: reflect the change immediately.
    const next = new Set(reacted);
    if (had) next.delete(kind);
    else next.add(kind);
    setReacted(next);
    setCount((c) => c + (had ? -1 : 1));
    setBusy(true);

    try {
      const state = had
        ? await apiDelete<ReactionState>(`/posts/${post.id}/reactions/${kind}`)
        : await apiPut<ReactionState>(`/posts/${post.id}/reactions/${kind}`);
      // Reconcile with the server's authoritative numbers.
      setCount(state.reaction_count);
      setReacted(new Set(state.viewer_reacted));
    } catch {
      // Roll back the optimistic change.
      setReacted(reacted);
      setCount(post.stats.reactions);
    } finally {
      setBusy(false);
    }
  }

  async function toggleBookmark() {
    if (!user || busy) return;
    const was = bookmarked;
    setBookmarked(!was);
    setBusy(true);
    try {
      if (was) await apiDelete(`/posts/${post.id}/bookmark`);
      else await apiPut(`/posts/${post.id}/bookmark`);
    } catch {
      setBookmarked(was);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex items-center gap-2">
      {KINDS.map(({ kind, glyph, label }) => {
        const active = reacted.has(kind);
        return (
          <button
            key={kind}
            type="button"
            aria-pressed={active}
            aria-label={label}
            disabled={!user}
            onClick={() => void toggleReaction(kind)}
            className={
              active
                ? "rounded-[--radius-control] bg-accent-subtle px-3 py-1.5 text-sm text-accent"
                : "rounded-[--radius-control] border border-border-strong px-3 py-1.5 text-sm text-text-muted hover:bg-surface-sunken disabled:opacity-50"
            }
          >
            <span aria-hidden>{glyph}</span>
          </button>
        );
      })}
      <span className="ml-1 text-sm text-text-subtle">{count}</span>

      <button
        type="button"
        aria-pressed={bookmarked}
        aria-label="Bookmark"
        disabled={!user}
        onClick={() => void toggleBookmark()}
        className={
          bookmarked
            ? "ml-3 rounded-[--radius-control] bg-accent-subtle px-3 py-1.5 text-sm text-accent"
            : "ml-3 rounded-[--radius-control] border border-border-strong px-3 py-1.5 text-sm text-text-muted hover:bg-surface-sunken disabled:opacity-50"
        }
      >
        <span aria-hidden>{bookmarked ? "★" : "☆"}</span> Save
      </button>
    </div>
  );
}
