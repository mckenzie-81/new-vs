import { PulseClient } from "./pulse-client";

export default async function PulsePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <PulseClient electionId={id} />;
}
