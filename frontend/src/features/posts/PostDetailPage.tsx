import { useEffect, useRef } from "react";
import { useParams } from "react-router-dom";

import { CommentSection } from "@/features/comments/CommentSection";
import { ReactionBar } from "@/features/reactions/ReactionBar";

import { tagClass } from "./tag-color";
import { recordView, usePostBySlug } from "./api";

/** Public reading page for a published post, rendered from the sanitized HTML
 * the backend produced. */
export function PostDetailPage() {
  const params = useParams<{ username: string; slug: string }>();
  // The route segment captures the leading "@" from /@username/slug URLs.
  const username = (params.username ?? "").replace(/^@/, "");
  const slug = params.slug ?? "";
  const { data: post, isPending, error } = usePostBySlug(username, slug);

  // Record one view per post once it loads. The ref guards against firing twice
  // for the same post (including React StrictMode's double effect invocation).
  const viewedRef = useRef<string | null>(null);
  useEffect(() => {
    if (post && viewedRef.current !== post.id) {
      viewedRef.current = post.id;
      recordView(post.id);
    }
  }, [post]);

  if (isPending) {
    return <div className="h-96 rounded-[--radius-card] bg-surface-raised" aria-busy />;
  }
  if (error) {
    return (
      <p role="alert" className="text-danger">
        Post not found.
      </p>
    );
  }

  return (
    <article className="mx-auto max-w-[68ch]">
      <h1 className="text-display font-bold leading-tight">{post.title}</h1>
      {post.subtitle && <p className="mt-2 text-xl text-text-muted">{post.subtitle}</p>}

      <div className="mt-4 flex items-center gap-2 text-sm text-text-subtle">
        {post.author.avatar_url && (
          <img src={post.author.avatar_url} alt="" className="h-8 w-8 rounded-full" />
        )}
        <span className="text-text-muted">{post.author.display_name}</span>
        <span aria-hidden>·</span>
        <span>{post.reading_minutes} min read</span>
      </div>

      {post.tags.length > 0 && (
        <ul className="mt-4 flex flex-wrap gap-2">
          {post.tags.map((t) => (
            <li key={t.name} className={`rounded-[--radius-tag] px-2 py-0.5 text-xs ${tagClass(t.name, t.color_key)}`}>
              #{t.name}
            </li>
          ))}
        </ul>
      )}

      {/* Safe: body_html is sanitized server-side before storage (bluemonday,
          docs/01-architecture §5). The client never renders unsanitized input. */}
      <div
        className="article mt-8"
        dangerouslySetInnerHTML={{ __html: post.body_html ?? "" }}
      />

      <div className="mt-8 border-t border-border-subtle pt-6">
        <ReactionBar post={post} />
      </div>

      <CommentSection postId={post.id} />
    </article>
  );
}
