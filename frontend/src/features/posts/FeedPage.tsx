import { useEffect, useRef } from "react";

import { useFeed } from "./api";
import { PostCard } from "./PostCard";

/** The home feed of published posts, loaded page by page on scroll. */
export function FeedPage() {
  const { data, isPending, error, fetchNextPage, hasNextPage, isFetchingNextPage } = useFeed();
  const sentinel = useRef<HTMLDivElement>(null);

  // Load the next page when the sentinel scrolls into view. This syncs React
  // with the browser's scroll position — a genuine external-system effect.
  useEffect(() => {
    const node = sentinel.current;
    if (!node || !hasNextPage) return;

    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting && !isFetchingNextPage) {
        void fetchNextPage();
      }
    });
    observer.observe(node);
    return () => {
      observer.disconnect();
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (isPending) {
    return <FeedSkeleton />;
  }

  if (error) {
    return (
      <p role="alert" className="text-danger">
        Could not load the feed. Is the API running?
      </p>
    );
  }

  const posts = data.pages.flatMap((page) => page.data);

  if (posts.length === 0) {
    return (
      <div className="rounded-[--radius-card] border border-border-subtle bg-surface-raised p-10 text-center">
        <p className="text-text-muted">No posts yet.</p>
        <p className="mt-1 text-sm text-text-subtle">Be the first to publish one.</p>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      {posts.map((post) => (
        <PostCard key={post.id} post={post} />
      ))}
      <div ref={sentinel} className="h-8" />
      {isFetchingNextPage && <p className="text-center text-sm text-text-subtle">Loading…</p>}
    </div>
  );
}

function FeedSkeleton() {
  return (
    <div className="space-y-5" aria-busy>
      {[0, 1, 2].map((i) => (
        <div key={i} className="h-40 rounded-[--radius-card] border border-border-subtle bg-surface-raised" />
      ))}
    </div>
  );
}
