import { useState } from "react";

import { tagClass } from "@/features/posts/tag-color";

interface TagInputProps {
  tags: string[];
  onChange: (tags: string[]) => void;
}

const MAX_TAGS = 4;
const TAG_PATTERN = /^[a-z0-9][a-z0-9-]{0,29}$/;

/** Chip-style tag input: type a tag, press Enter or comma to add it. Enforces
 * the same 4-tag limit and format the backend does, for instant feedback. */
export function TagInput({ tags, onChange }: TagInputProps) {
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);

  function commit() {
    const name = draft.trim().toLowerCase();
    setDraft("");
    if (name === "") return;
    if (tags.includes(name)) return;
    if (tags.length >= MAX_TAGS) {
      setError("At most 4 tags");
      return;
    }
    if (!TAG_PATTERN.test(name)) {
      setError("Tags are lowercase letters, digits or hyphens");
      return;
    }
    setError(null);
    onChange([...tags, name]);
  }

  return (
    <div>
      <div className="flex flex-wrap items-center gap-2 rounded-[--radius-control] border border-border-strong px-2 py-1.5">
        {tags.map((name) => (
          <span
            key={name}
            className={`flex items-center gap-1 rounded-[--radius-tag] px-2 py-0.5 text-xs ${tagClass(name)}`}
          >
            #{name}
            <button
              type="button"
              aria-label={`Remove ${name}`}
              onClick={() => {
                onChange(tags.filter((t) => t !== name));
              }}
            >
              ×
            </button>
          </span>
        ))}
        {tags.length < MAX_TAGS && (
          <input
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === ",") {
                e.preventDefault();
                commit();
              } else if (e.key === "Backspace" && draft === "" && tags.length > 0) {
                onChange(tags.slice(0, -1));
              }
            }}
            onBlur={commit}
            placeholder={tags.length === 0 ? "Add up to 4 tags…" : ""}
            className="min-w-24 flex-1 bg-transparent text-sm outline-none"
          />
        )}
      </div>
      {error && <p className="mt-1 text-xs text-danger">{error}</p>}
    </div>
  );
}
