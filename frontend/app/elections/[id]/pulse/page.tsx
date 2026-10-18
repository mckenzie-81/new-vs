import { PulseClient } from "./pulse-client";

// Always serves a real, incoming request for a specific election id -
// nothing here can be prerendered ahead of time. Same reasoning as
// app/vote/[token]/page.tsx.
export const instant = false;

export default async function PulsePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <PulseClient electionId={id} />;
}
