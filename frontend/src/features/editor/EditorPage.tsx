import { useParams } from "react-router-dom";

import { usePost } from "@/features/posts/api";

import { EditorForm } from "./EditorForm";

const EMPTY = { title: "", subtitle: "", bodyMarkdown: "", tags: [], bodyHTML: "" };

/** Editor route. For /new it opens a blank form; for /edit/:id it loads the post
 * first, then renders the form seeded with its content. */
export function EditorPage() {
  const { id } = useParams<{ id: string }>();

  if (!id) {
    return <EditorForm key="new" initial={EMPTY} />;
  }
  return <EditPageLoader id={id} />;
}

function EditPageLoader({ id }: { id: string }) {
  const { data: post, isPending, error } = usePost(id);

  if (isPending) {
    return <div className="h-96 rounded-[--radius-card] bg-surface-raised" aria-busy />;
  }
  if (error) {
    return (
      <p role="alert" className="text-danger">
        Could not load this draft.
      </p>
    );
  }

  return (
    <EditorForm
      key={id}
      postId={id}
      initial={{
        title: post.title,
        subtitle: post.subtitle ?? "",
        bodyMarkdown: post.body_markdown ?? "",
        tags: post.tags.map((t) => t.name),
        bodyHTML: post.body_html ?? "",
      }}
    />
  );
}
