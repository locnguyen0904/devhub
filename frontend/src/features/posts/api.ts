import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  apiDelete,
  apiGet,
  apiPatch,
  apiPost,
} from "@/shared/api/client";
import type {
  CreatePostBody,
  Post,
  PostFeed,
  PostList,
  SearchResults,
  TagChip,
  UpdatePostBody,
} from "@/shared/types";

const FEED_KEY = ["feed"] as const;
const MY_POSTS_KEY = ["my-posts"] as const;

export type FeedSort = "latest" | "hot";

/**
 * The published feed, paginated by cursor for infinite scroll. Hot returns a
 * single page (its ranking has no stable cursor), so has_more is false and the
 * scroll naturally stops. Tag filters the latest feed.
 */
export function useFeed(sort: FeedSort = "latest", tag?: string) {
  return useInfiniteQuery({
    queryKey: [...FEED_KEY, sort, tag ?? ""],
    queryFn: ({ pageParam, signal }) => {
      const params = new URLSearchParams({ limit: "20", sort, cursor: pageParam });
      if (tag) params.set("tag", tag);
      return apiGet<PostFeed>(`/posts?${params.toString()}`, signal);
    },
    initialPageParam: "",
    getNextPageParam: (last) =>
      last.page.has_more ? last.page.next_cursor : undefined,
  });
}

/** Full-text search. Disabled until the query is at least 2 characters. */
export function useSearch(query: string) {
  const trimmed = query.trim();
  return useQuery({
    queryKey: ["search", trimmed],
    queryFn: ({ signal }) =>
      apiGet<SearchResults>(`/search?q=${encodeURIComponent(trimmed)}`, signal),
    enabled: trimmed.length >= 2,
  });
}

/** A single post by id, for the editor. */
export function usePost(id: string | undefined) {
  return useQuery({
    queryKey: ["post", id],
    queryFn: ({ signal }) => apiGet<Post>(`/posts/${id ?? ""}`, signal),
    enabled: Boolean(id),
  });
}

/** A published post by its public username/slug URL. */
export function usePostBySlug(username: string, slug: string) {
  return useQuery({
    queryKey: ["post-by-slug", username, slug],
    queryFn: ({ signal }) =>
      apiGet<Post>(`/posts/by-slug/${username}/${slug}`, signal),
  });
}

/** The posts the current user has bookmarked, newest saved first. */
export function useBookmarks() {
  return useQuery({
    queryKey: ["bookmarks"],
    queryFn: ({ signal }) => apiGet<PostList>("/me/bookmarks", signal),
  });
}

/** The current user's posts, filtered by status. */
export function useMyPosts(status: "all" | "draft" | "published") {
  return useQuery({
    queryKey: [...MY_POSTS_KEY, status],
    queryFn: ({ signal }) =>
      apiGet<PostList>(`/me/posts?status=${status}`, signal),
  });
}

export function useCreatePost() {
  return useMutation({
    mutationFn: (body: CreatePostBody) => apiPost<Post>("/posts", body),
  });
}

export function useUpdatePost(id: string) {
  return useMutation({
    mutationFn: (body: UpdatePostBody) => apiPatch<Post>(`/posts/${id}`, body),
  });
}

export function usePublishPost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<Post>(`/posts/${id}/publish`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: FEED_KEY });
      void queryClient.invalidateQueries({ queryKey: MY_POSTS_KEY });
    },
  });
}

/** Records a view, fire-and-forget. Failures are ignored — a missed view count
 * is not worth surfacing to the reader. */
export function recordView(postId: string): void {
  void apiPost(`/posts/${postId}/views`).catch(() => {
    // intentionally ignored
  });
}

export function useDeletePost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete<unknown>(`/posts/${id}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: MY_POSTS_KEY });
    },
  });
}

/** Tag autocomplete for the editor's tag input. */
export function searchTags(query: string, signal?: AbortSignal): Promise<TagChip[]> {
  return apiGet<{ tags: TagChip[] }>(
    `/tags?q=${encodeURIComponent(query)}&limit=6`,
    signal,
  ).then((r) => r.tags);
}
