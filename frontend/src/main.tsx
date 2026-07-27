// This is the app entry, not a fast-refreshable component module, so the lazy
// EditorPage const alongside the render call is fine.
/* eslint-disable react-refresh/only-export-components */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Route, Routes } from "react-router-dom";

import { App } from "@/app/App";
import { RequireAuth } from "@/app/RequireAuth";
import { FeedPage } from "@/features/posts/FeedPage";
import { MyPostsPage } from "@/features/posts/MyPostsPage";
import { PostDetailPage } from "@/features/posts/PostDetailPage";
import { SearchPage } from "@/features/posts/SearchPage";
import { TagPage } from "@/features/posts/TagPage";

import "./app/theme.css";

// The editor pulls in CodeMirror, which is large and only needed on /new and
// /edit — lazy-load it so the feed and reading pages stay light.
const EditorPage = lazy(() =>
  import("@/features/editor/EditorPage").then((m) => ({ default: m.EditorPage })),
);

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, refetchOnWindowFocus: false },
  },
});

const root = document.getElementById("root");
if (!root) throw new Error("missing #root element in index.html");

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<App />}>
            <Route index element={<FeedPage />} />
            <Route path="search" element={<SearchPage />} />
            <Route path="tags/:name" element={<TagPage />} />
            <Route
              path="new"
              element={
                <RequireAuth>
                  <Suspense fallback={null}>
                    <EditorPage />
                  </Suspense>
                </RequireAuth>
              }
            />
            <Route
              path="edit/:id"
              element={
                <RequireAuth>
                  <Suspense fallback={null}>
                    <EditorPage />
                  </Suspense>
                </RequireAuth>
              }
            />
            <Route
              path="me/posts"
              element={
                <RequireAuth>
                  <MyPostsPage />
                </RequireAuth>
              }
            />
            <Route path=":username/:slug" element={<PostDetailPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
