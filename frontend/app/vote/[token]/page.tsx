import { VoteClient } from "./vote-client";

export default async function VotePage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;
  return <VoteClient token={token} />;
}
