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
  TagChip,
  UpdatePostBody,
} from "@/shared/types";

const FEED_KEY = ["feed"] as const;
const MY_POSTS_KEY = ["my-posts"] as const;

/** The published feed, one page per cursor, for infinite scroll. */
export function useFeed() {
  return useInfiniteQuery({
    queryKey: FEED_KEY,
    queryFn: ({ pageParam, signal }) =>
      apiGet<PostFeed>(`/posts?limit=20&cursor=${pageParam}`, signal),
    initialPageParam: "",
    getNextPageParam: (last) =>
      last.page.has_more ? last.page.next_cursor : undefined,
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
