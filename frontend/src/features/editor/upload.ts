import { apiPost } from "@/shared/api/client";
import type { PresignResult } from "@/shared/types";

const MAX_BYTES = 5 * 1024 * 1024;
const ALLOWED = new Set(["image/png", "image/jpeg", "image/webp", "image/gif"]);

/**
 * Uploads an image and returns its public URL. The file goes straight to
 * storage via a presigned PUT — the PUT is a raw fetch, not the API client, so
 * it carries no auth header and no /api prefix.
 */
export async function uploadImage(file: File): Promise<string> {
  if (!ALLOWED.has(file.type)) {
    throw new Error("Only PNG, JPEG, WebP or GIF images are allowed");
  }
  if (file.size > MAX_BYTES) {
    throw new Error("Image must be 5 MB or smaller");
  }

  const presign = await apiPost<PresignResult>("/uploads/presign", {
    content_type: file.type,
    size_bytes: file.size,
  });

  const put = await fetch(presign.upload_url, {
    method: "PUT",
    headers: { "Content-Type": file.type },
    body: file,
  });
  if (!put.ok) {
    throw new Error(`Upload failed (${String(put.status)})`);
  }

  return presign.public_url;
}
