import { VoteClient } from "./vote-client";

// This route only ever serves a real, incoming request - the token is
// per-voter and never knowable ahead of time, so there's nothing here to
// prerender or shell statically. Awaiting `params` is what Next 16's
// instant-navigation validator (Cache Components) flags as a "blocking
// prerender"; opting out is correct for a route that's always dynamic by
// nature, not a workaround for a bug.
export const instant = false;

export default async function VotePage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;
  return <VoteClient token={token} />;
}
