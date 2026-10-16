import { Suspense } from "react";
import AcceptInviteClient from "./accept-client";

export default function AcceptInvitePage() {
  return (
    <Suspense>
      <AcceptInviteClient />
    </Suspense>
  );
}
