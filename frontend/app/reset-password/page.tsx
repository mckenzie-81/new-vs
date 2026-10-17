import { Suspense } from "react";
import ResetPasswordClient from "./reset-client";

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordClient />
    </Suspense>
  );
}
