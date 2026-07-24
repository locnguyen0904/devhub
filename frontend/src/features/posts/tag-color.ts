/** The 8 palette keys, in the order the tags table CHECK constrains them. */
const TAG_KEYS = [
  "blue",
  "violet",
  "emerald",
  "amber",
  "rose",
  "cyan",
  "orange",
  "teal",
] as const;

/**
 * Returns the CSS class for a tag. When the backend has no assigned color_key,
 * derive one from the name so the same tag always gets the same colour. The
 * hash is stable and only depends on the name — matching the backend's intent.
 */
export function tagClass(name: string, colorKey?: string | null): string {
  const key = colorKey ?? TAG_KEYS[hash(name) % TAG_KEYS.length] ?? "blue";
  return `tag tag-${key}`;
}

function hash(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}
