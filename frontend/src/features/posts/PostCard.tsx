import { Link } from "react-router-dom";

import type { Post } from "@/shared/types";

import { tagClass } from "./tag-color";

interface PostCardProps {
  post: Post;
}

/** A post as it appears in the feed: cover, title, tags, author, stats. */
export function PostCard({ post }: PostCardProps) {
  return (
    <article className="rounded-[--radius-card] border border-border-subtle bg-surface-raised shadow-[--shadow-card]">
      {post.cover_image_url && (
        <Link to={post.url}>
          <img
            src={post.cover_image_url}
            alt=""
            className="aspect-[2/1] w-full rounded-t-[--radius-card] object-cover"
          />
        </Link>
      )}
      <div className="p-5">
        <div className="mb-2 flex items-center gap-2 text-sm text-text-subtle">
          {post.author.avatar_url && (
            <img src={post.author.avatar_url} alt="" className="h-6 w-6 rounded-full" />
          )}
          <span>{post.author.display_name}</span>
          {post.published_at && (
            <>
              <span aria-hidden>·</span>
              <time dateTime={post.published_at}>{formatDate(post.published_at)}</time>
            </>
          )}
          <span aria-hidden>·</span>
          <span>{post.reading_minutes} min read</span>
        </div>

        <Link to={post.url}>
          <h2 className="text-xl font-semibold hover:text-accent">{post.title}</h2>
        </Link>
        {post.subtitle && <p className="mt-1 text-text-muted">{post.subtitle}</p>}
        {post.excerpt && <p className="mt-2 line-clamp-2 text-sm text-text-subtle">{post.excerpt}</p>}

        {post.tags.length > 0 && (
          <ul className="mt-3 flex flex-wrap gap-2">
            {post.tags.map((t) => (
              <li key={t.name} className={`rounded-[--radius-tag] px-2 py-0.5 text-xs ${tagClass(t.name, t.color_key)}`}>
                #{t.name}
              </li>
            ))}
          </ul>
        )}

        <div className="mt-3 flex gap-4 text-sm text-text-subtle">
          <span>♥ {post.stats.reactions}</span>
          <span>💬 {post.stats.comments}</span>
        </div>
      </div>
    </article>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("vi-VN", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}
