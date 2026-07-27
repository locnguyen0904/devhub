import { useState } from "react";

import { useFeed, type FeedSort } from "./api";
import { FeedList } from "./FeedList";

const SORTS: { key: FeedSort; label: string }[] = [
  { key: "latest", label: "Latest" },
  { key: "hot", label: "Hot" },
];

/** The home feed, switchable between newest and the ranked "hot" order. */
export function FeedPage() {
  const [sort, setSort] = useState<FeedSort>("latest");
  const query = useFeed(sort);

  return (
    <div>
      <div className="mb-5 flex gap-1 rounded-[--radius-control] border border-border-subtle p-1">
        {SORTS.map((s) => (
          <button
            key={s.key}
            type="button"
            onClick={() => {
              setSort(s.key);
            }}
            className={
              sort === s.key
                ? "rounded-[--radius-control] bg-accent-subtle px-4 py-1 text-sm font-medium text-accent"
                : "px-4 py-1 text-sm text-text-muted"
            }
          >
            {s.label}
          </button>
        ))}
      </div>
      <FeedList query={query} emptyMessage="No posts yet. Be the first to publish one." />
    </div>
  );
}
