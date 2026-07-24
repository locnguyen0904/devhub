import { markdown } from "@codemirror/lang-markdown";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import { apiPatch, apiPost } from "@/shared/api/client";
import type { Post } from "@/shared/types";

import { TagInput } from "./TagInput";
import { uploadImage } from "./upload";

const AUTOSAVE_DELAY = 1000;

interface EditorFormProps {
  postId?: string;
  initial: {
    title: string;
    subtitle: string;
    bodyMarkdown: string;
    tags: string[];
    bodyHTML: string;
  };
}

type SaveState = "idle" | "saving" | "saved" | "error";

/** The post editor: markdown on the left, live preview (the last saved HTML) on
 * the right. Drafts autosave; the first save of a new post creates it. */
export function EditorForm({ postId: initialID, initial }: EditorFormProps) {
  const navigate = useNavigate();

  const [title, setTitle] = useState(initial.title);
  const [subtitle, setSubtitle] = useState(initial.subtitle);
  const [body, setBody] = useState(initial.bodyMarkdown);
  const [tags, setTags] = useState<string[]>(initial.tags);
  const [previewHTML, setPreviewHTML] = useState(initial.bodyHTML);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [publishing, setPublishing] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  // The CodeMirror view, captured on create, so an uploaded image can be
  // inserted at the caret.
  const viewRef = useRef<EditorView | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // postId is held in a ref so the debounced save closure always sees the
  // latest id after a new post's first save, without re-arming the timer.
  const postIdRef = useRef(initialID);
  // Snapshot of what was last persisted, so an unchanged debounce tick is a no-op.
  const savedRef = useRef(serialize(initial.title, initial.subtitle, initial.bodyMarkdown, initial.tags));

  const save = useCallback(async () => {
    const snapshot = serialize(title, subtitle, body, tags);
    if (snapshot === savedRef.current) return;
    if (title.trim() === "") return; // nothing worth saving yet

    setSaveState("saving");
    try {
      let post: Post;
      if (postIdRef.current) {
        post = await apiPatch<Post>(`/posts/${postIdRef.current}`, {
          title,
          subtitle: subtitle || undefined,
          body_markdown: body,
        });
      } else {
        post = await apiPost<Post>("/posts", {
          title,
          subtitle: subtitle || undefined,
          body_markdown: body,
          tags,
        });
        postIdRef.current = post.id;
        // Move the URL to the edit route without a reload, so a refresh reopens
        // the draft rather than a blank new-post form.
        void navigate(`/edit/${post.id}`, { replace: true });
      }
      savedRef.current = snapshot;
      setPreviewHTML(post.body_html ?? "");
      setSaveState("saved");
    } catch {
      setSaveState("error");
    }
  }, [title, subtitle, body, tags, navigate]);

  // Debounced autosave: every edit re-arms a timer; a pause of AUTOSAVE_DELAY
  // triggers the save. Syncing to the server on a timer is a real effect.
  useEffect(() => {
    const timer = setTimeout(() => {
      void save();
    }, AUTOSAVE_DELAY);
    return () => {
      clearTimeout(timer);
    };
  }, [save]);

  async function publish() {
    await save();
    const id = postIdRef.current;
    if (!id) return;
    setPublishing(true);
    try {
      const post = await apiPost<Post>(`/posts/${id}/publish`);
      void navigate(post.url);
    } catch {
      setPublishing(false);
    }
  }

  // Insert text at the caret by dispatching a CodeMirror transaction; that fires
  // onChange, so React state stays in sync.
  function insertAtCursor(text: string) {
    const view = viewRef.current;
    if (!view) return;
    const pos = view.state.selection.main.head;
    view.dispatch({
      changes: { from: pos, insert: text },
      selection: { anchor: pos + text.length },
    });
    view.focus();
  }

  async function handleFiles(files: FileList | null) {
    const file = files?.[0];
    if (!file) return;
    setUploadError(null);
    setUploading(true);
    try {
      const url = await uploadImage(file);
      insertAtCursor(`\n![${file.name}](${url})\n`);
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <SaveIndicator state={saveState} />
        <button
          type="button"
          onClick={() => void publish()}
          disabled={publishing || title.trim() === ""}
          className="rounded-[--radius-control] bg-accent px-4 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent-hover disabled:opacity-50"
        >
          Publish
        </button>
      </div>

      <input
        value={title}
        onChange={(e) => {
          setTitle(e.target.value);
        }}
        placeholder="Post title"
        className="w-full bg-transparent text-3xl font-bold outline-none placeholder:text-text-subtle"
      />
      <input
        value={subtitle}
        onChange={(e) => {
          setSubtitle(e.target.value);
        }}
        placeholder="Subtitle (optional)"
        className="mt-2 w-full bg-transparent text-lg text-text-muted outline-none placeholder:text-text-subtle"
      />

      <div className="mt-3">
        <TagInput tags={tags} onChange={setTags} />
      </div>

      <div className="mt-3 flex items-center gap-3">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          className="hidden"
          onChange={(e) => {
            void handleFiles(e.target.files);
            e.target.value = ""; // allow re-selecting the same file
          }}
        />
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          className="rounded-[--radius-control] border border-border-strong px-3 py-1 text-sm text-text-muted hover:bg-surface-sunken disabled:opacity-50"
        >
          {uploading ? "Uploading…" : "Insert image"}
        </button>
        {uploadError && <span className="text-sm text-danger">{uploadError}</span>}
      </div>

      <div className="mt-2 grid gap-4 md:grid-cols-2">
        {/* Drop an image anywhere on the editor to upload and insert it. */}
        <div
          onDrop={(e) => {
            e.preventDefault();
            void handleFiles(e.dataTransfer.files);
          }}
          onDragOver={(e) => {
            e.preventDefault();
          }}
        >
          <CodeMirror
            value={body}
            onChange={setBody}
            onCreateEditor={(view) => {
              viewRef.current = view;
            }}
            extensions={[markdown(), EditorView.lineWrapping]}
            basicSetup={{ lineNumbers: false, foldGutter: false }}
            placeholder="Write your post in Markdown… drop an image to upload it."
            className="rounded-[--radius-card] border border-border-subtle text-sm"
            height="60vh"
          />
        </div>
        <div className="overflow-y-auto rounded-[--radius-card] border border-border-subtle bg-surface-raised p-5" style={{ height: "60vh" }}>
          {previewHTML ? (
            // Safe: previewHTML is the sanitized body_html the backend returned.
            <div className="article" dangerouslySetInnerHTML={{ __html: previewHTML }} />
          ) : (
            <p className="text-sm text-text-subtle">Preview appears after the first save.</p>
          )}
        </div>
      </div>
    </div>
  );
}

function SaveIndicator({ state }: { state: SaveState }) {
  const label: Record<SaveState, string> = {
    idle: "Draft",
    saving: "Saving…",
    saved: "Saved",
    error: "Save failed",
  };
  return (
    <span className={state === "error" ? "text-sm text-danger" : "text-sm text-text-subtle"}>
      {label[state]}
    </span>
  );
}

function serialize(title: string, subtitle: string, body: string, tags: string[]): string {
  return JSON.stringify([title, subtitle, body, tags]);
}
