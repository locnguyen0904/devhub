import { Link, useSearchParams } from "react-router-dom";

import { useSearch } from "./api";
import { tagClass } from "./tag-color";

/** Search results page, reading the query from ?q= and rendering each hit with
 * its highlighted snippet. */
export function SearchPage() {
  const [params] = useSearchParams();
  const q = params.get("q") ?? "";
  const { data, isPending, error, isFetching } = useSearch(q);

  return (
    <div>
      <h1 className="mb-1 text-h1 font-bold">Search</h1>
      {q ? (
        <p className="mb-6 text-text-muted">
          Results for “<span className="text-text-primary">{q}</span>”
        </p>
      ) : (
        <p className="mb-6 text-text-muted">Type at least 2 characters to search.</p>
      )}

      {q.trim().length >= 2 && (isPending || isFetching) ? (
        <div className="h-40 rounded-[--radius-card] bg-surface-raised" aria-busy />
      ) : error ? (
        <p role="alert" className="text-danger">
          Search failed. Please try again.
        </p>
      ) : data && data.data.length > 0 ? (
        <ul className="space-y-4">
          {data.data.map((hit) => (
            <li
              key={hit.post.id}
              className="rounded-[--radius-card] border border-border-subtle bg-surface-raised p-5"
            >
              <Link to={hit.post.url} className="text-lg font-semibold hover:text-accent">
                {hit.post.title}
              </Link>
              <div className="mt-1 flex flex-wrap gap-2">
                {hit.post.tags.map((t) => (
                  <span
                    key={t.name}
                    className={`rounded-[--radius-tag] px-2 py-0.5 text-xs ${tagClass(t.name, t.color_key)}`}
                  >
                    #{t.name}
                  </span>
                ))}
              </div>
              {/* Safe: the headline is a server-built snippet whose only markup
                  is <b> around matched terms. */}
              <p
                className="mt-2 text-sm text-text-muted [&_b]:font-semibold [&_b]:text-text-primary"
                dangerouslySetInnerHTML={{ __html: hit.headline }}
              />
            </li>
          ))}
        </ul>
      ) : (
        q.trim().length >= 2 && <p className="text-text-muted">No posts match “{q}”.</p>
      )}
    </div>
  );
}
