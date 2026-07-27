import { useParams } from "react-router-dom";

import { useFeed } from "./api";
import { FeedList } from "./FeedList";

/** Lists published posts carrying one tag. */
export function TagPage() {
  const { name } = useParams<{ name: string }>();
  const tag = name ?? "";
  const query = useFeed("latest", tag);

  return (
    <div>
      <h1 className="mb-5 text-h1 font-bold">
        <span className="text-text-subtle">#</span>
        {tag}
      </h1>
      <FeedList query={query} emptyMessage={`No posts tagged #${tag} yet.`} />
    </div>
  );
}
