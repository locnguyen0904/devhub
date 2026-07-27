import { useState } from "react";

import { useSession } from "@/features/auth/api";
import type { Comment } from "@/shared/types";

import {
  useComments,
  useCreateComment,
  useDeleteComment,
  useUpdateComment,
} from "./api";

interface CommentSectionProps {
  postId: string;
}

/** The comment thread under a post: a top-level form plus the tree. */
export function CommentSection({ postId }: CommentSectionProps) {
  const { data, isPending, error } = useComments(postId);
  const { user } = useSession();
  const create = useCreateComment(postId);

  return (
    <section className="mt-12">
      <h2 className="mb-4 text-h2 font-semibold">Comments</h2>

      {user ? (
        <CommentForm
          submitting={create.isPending}
          onSubmit={(body) => {
            create.mutate({ bodyMarkdown: body });
          }}
        />
      ) : (
        <p className="text-sm text-text-subtle">Sign in to comment.</p>
      )}

      {isPending ? (
        <div className="mt-6 h-20 rounded-[--radius-card] bg-surface-raised" aria-busy />
      ) : error ? (
        <p role="alert" className="mt-6 text-danger">
          Could not load comments.
        </p>
      ) : data.data.length === 0 ? (
        <p className="mt-6 text-text-subtle">No comments yet.</p>
      ) : (
        <ul className="mt-6 space-y-5">
          {data.data.map((c) => (
            <CommentItem key={c.id} postId={postId} comment={c} currentUser={user?.username} />
          ))}
        </ul>
      )}
    </section>
  );
}

interface CommentItemProps {
  postId: string;
  comment: Comment;
  currentUser?: string;
  isReply?: boolean;
}

function CommentItem({ postId, comment, currentUser, isReply }: CommentItemProps) {
  const [replying, setReplying] = useState(false);
  const [editing, setEditing] = useState(false);
  const create = useCreateComment(postId);
  const update = useUpdateComment(postId);
  const remove = useDeleteComment(postId);

  const isOwn = !comment.deleted && comment.author?.username === currentUser;

  return (
    <li>
      <div className="rounded-[--radius-card] border border-border-subtle bg-surface-raised p-4">
        {comment.deleted ? (
          <p className="text-sm italic text-text-subtle">[deleted]</p>
        ) : (
          <>
            <div className="mb-1 flex items-center gap-2 text-sm">
              {comment.author?.avatar_url && (
                <img src={comment.author.avatar_url} alt="" className="h-5 w-5 rounded-full" />
              )}
              <span className="font-medium">{comment.author?.display_name}</span>
              <span className="text-text-subtle">{formatDate(comment.created_at)}</span>
            </div>

            {editing ? (
              <CommentForm
                initial={comment.body_markdown ?? ""}
                submitting={update.isPending}
                onCancel={() => {
                  setEditing(false);
                }}
                onSubmit={(body) => {
                  update.mutate(
                    { id: comment.id, bodyMarkdown: body },
                    { onSuccess: () => { setEditing(false); } },
                  );
                }}
              />
            ) : (
              <div
                className="article text-sm"
                // Safe: body_html is sanitized server-side before storage.
                dangerouslySetInnerHTML={{ __html: comment.body_html ?? "" }}
              />
            )}

            {!editing && (
              <div className="mt-2 flex gap-3 text-xs text-text-subtle">
                {!isReply && currentUser && (
                  <button type="button" onClick={() => { setReplying((v) => !v); }} className="hover:text-text-primary">
                    Reply
                  </button>
                )}
                {isOwn && (
                  <>
                    <button type="button" onClick={() => { setEditing(true); }} className="hover:text-text-primary">
                      Edit
                    </button>
                    <button type="button" onClick={() => { remove.mutate(comment.id); }} className="hover:text-danger">
                      Delete
                    </button>
                  </>
                )}
              </div>
            )}
          </>
        )}
      </div>

      {replying && (
        <div className="ml-6 mt-2">
          <CommentForm
            submitting={create.isPending}
            onCancel={() => { setReplying(false); }}
            onSubmit={(body) => {
              create.mutate(
                { bodyMarkdown: body, parentId: comment.id },
                { onSuccess: () => { setReplying(false); } },
              );
            }}
          />
        </div>
      )}

      {comment.replies.length > 0 && (
        <ul className="ml-6 mt-3 space-y-3 border-l border-border-subtle pl-4">
          {comment.replies.map((r) => (
            <CommentItem key={r.id} postId={postId} comment={r} currentUser={currentUser} isReply />
          ))}
        </ul>
      )}
    </li>
  );
}

interface CommentFormProps {
  initial?: string;
  submitting: boolean;
  onSubmit: (body: string) => void;
  onCancel?: () => void;
}

function CommentForm({ initial = "", submitting, onSubmit, onCancel }: CommentFormProps) {
  const [body, setBody] = useState(initial);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        if (body.trim()) onSubmit(body.trim());
      }}
    >
      <textarea
        value={body}
        onChange={(e) => { setBody(e.target.value); }}
        rows={3}
        placeholder="Write a comment (Markdown supported)…"
        className="w-full rounded-[--radius-control] border border-border-strong bg-surface p-2 text-sm outline-none focus-visible:border-accent"
      />
      <div className="mt-2 flex gap-2">
        <button
          type="submit"
          disabled={submitting || body.trim() === ""}
          className="rounded-[--radius-control] bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent-hover disabled:opacity-50"
        >
          {submitting ? "Posting…" : "Post"}
        </button>
        {onCancel && (
          <button type="button" onClick={onCancel} className="px-3 py-1.5 text-sm text-text-muted">
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("vi-VN", { day: "numeric", month: "short", year: "numeric" });
}
