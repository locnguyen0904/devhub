import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiDelete, apiGet, apiPatch, apiPost } from "@/shared/api/client";
import type { Comment, CommentTree } from "@/shared/types";

function key(postId: string) {
  return ["comments", postId] as const;
}

/** The comment tree for a post. Public — visible to anyone reading the post. */
export function useComments(postId: string) {
  return useQuery({
    queryKey: key(postId),
    queryFn: ({ signal }) => apiGet<CommentTree>(`/posts/${postId}/comments`, signal),
  });
}

interface CreateArgs {
  bodyMarkdown: string;
  parentId?: string;
}

export function useCreateComment(postId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (args: CreateArgs) =>
      apiPost<Comment>(`/posts/${postId}/comments`, {
        body_markdown: args.bodyMarkdown,
        parent_id: args.parentId,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: key(postId) });
    },
  });
}

export function useUpdateComment(postId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (args: { id: string; bodyMarkdown: string }) =>
      apiPatch<Comment>(`/comments/${args.id}`, { body_markdown: args.bodyMarkdown }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: key(postId) });
    },
  });
}

export function useDeleteComment(postId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete<unknown>(`/comments/${id}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: key(postId) });
    },
  });
}
