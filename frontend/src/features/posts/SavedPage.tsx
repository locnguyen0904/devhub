import { Link } from "react-router-dom";

import { useBookmarks } from "./api";
import { PostCard } from "./PostCard";

/** The reader's saved posts. */
export function SavedPage() {
  const { data, isPending, error } = useBookmarks();

  return (
    <div>
      <h1 className="mb-6 text-h1 font-bold">Saved</h1>

      {isPending ? (
        <div className="h-40 rounded-[--radius-card] bg-surface-raised" aria-busy />
      ) : error ? (
        <p role="alert" className="text-danger">
          Could not load your saved posts.
        </p>
      ) : data.data.length === 0 ? (
        <p className="text-text-muted">
          Nothing saved yet.{" "}
          <Link to="/" className="text-accent underline">
            Browse the feed
          </Link>
          .
        </p>
      ) : (
        <div className="space-y-5">
          {data.data.map((post) => (
            <PostCard key={post.id} post={post} />
          ))}
        </div>
      )}
    </div>
  );
}
