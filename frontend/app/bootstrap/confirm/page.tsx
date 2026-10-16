import { Suspense } from "react";
import ConfirmClient from "./confirm-client";

export default function ConfirmPage() {
  return (
    <Suspense>
      <ConfirmClient />
    </Suspense>
  );
}
