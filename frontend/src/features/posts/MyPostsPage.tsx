import { useState } from "react";
import { Link } from "react-router-dom";

import { useDeletePost, useMyPosts } from "./api";

type Filter = "all" | "draft" | "published";

const FILTERS: Filter[] = ["all", "draft", "published"];

/** The author's own posts, filterable by status, with quick edit/delete. */
export function MyPostsPage() {
  const [filter, setFilter] = useState<Filter>("all");
  const { data, isPending, error } = useMyPosts(filter);
  const deletePost = useDeletePost();

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-h1 font-bold">My posts</h1>
        <div className="flex gap-1 rounded-[--radius-control] border border-border-subtle p-1">
          {FILTERS.map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => {
                setFilter(f);
              }}
              className={
                filter === f
                  ? "rounded-[--radius-control] bg-accent-subtle px-3 py-1 text-sm capitalize text-accent"
                  : "px-3 py-1 text-sm capitalize text-text-muted"
              }
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {isPending ? (
        <div className="h-40 rounded-[--radius-card] bg-surface-raised" aria-busy />
      ) : error ? (
        <p role="alert" className="text-danger">
          Could not load your posts.
        </p>
      ) : data.data.length === 0 ? (
        <p className="text-text-muted">
          No {filter === "all" ? "" : filter} posts yet.{" "}
          <Link to="/new" className="text-accent underline">
            Write one
          </Link>
          .
        </p>
      ) : (
        <ul className="divide-y divide-border-subtle rounded-[--radius-card] border border-border-subtle bg-surface-raised">
          {data.data.map((post) => (
            <li key={post.id} className="flex items-center justify-between gap-4 p-4">
              <div className="min-w-0">
                <p className="truncate font-medium">{post.title}</p>
                <span
                  className={
                    post.status === "published"
                      ? "text-xs text-success"
                      : "text-xs text-text-subtle"
                  }
                >
                  {post.status}
                </span>
              </div>
              <div className="flex shrink-0 gap-3 text-sm">
                <Link to={`/edit/${post.id}`} className="text-accent hover:underline">
                  Edit
                </Link>
                <button
                  type="button"
                  onClick={() => {
                    deletePost.mutate(post.id);
                  }}
                  className="text-danger hover:underline"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
