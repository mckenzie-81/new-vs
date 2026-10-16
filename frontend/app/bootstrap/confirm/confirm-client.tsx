"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ApiError, confirmBootstrap } from "@/lib/api";
import { CenteredCard, CenteredMessage } from "@/components/layout/centered-card";

type ConfirmState = "confirming" | "active" | "invalid" | "expired" | "error";

const content: Record<
  Exclude<ConfirmState, "confirming">,
  { heading: string; body: string }
> = {
  active: {
    heading: "Account activated",
    body: "The administrator account is ready. You'll be able to sign in once the sign-in page is in place.",
  },
  invalid: {
    heading: "Invalid link",
    body: "This verification link isn't valid. Double-check the link in your email, or start the setup process again.",
  },
  expired: {
    heading: "Link expired",
    body: "This verification link has expired. Start the setup process again to get a new one.",
  },
  error: {
    heading: "Something went wrong",
    body: "We couldn't confirm this account. Please try again in a moment.",
  },
};

export default function ConfirmClient() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const [state, setState] = useState<ConfirmState>(token ? "confirming" : "invalid");
  const requestedToken = useRef<string | null>(null);

  useEffect(() => {
    if (!token) return;

    // Guards against React Strict Mode's intentional double-invoke of
    // effects in development, which would otherwise fire this request
    // twice. The server tolerates a token being confirmed more than once
    // regardless, but there's no reason to send a redundant request when
    // we can tell it's about to happen.
    if (requestedToken.current === token) return;
    requestedToken.current = token;

    confirmBootstrap(token)
      .then(() => setState("active"))
      .catch((err) => {
        if (err instanceof ApiError && err.status === 400) {
          setState("invalid");
        } else if (err instanceof ApiError && err.status === 410) {
          setState("expired");
        } else {
          setState("error");
        }
      });
  }, [token]);

  if (state === "confirming") {
    return <CenteredMessage>Confirming your account...</CenteredMessage>;
  }

  const { heading, body } = content[state];

  return (
    <CenteredCard>
      <div className="text-center">
        <h1 className="mb-2 text-xl font-semibold tracking-tight">{heading}</h1>
        <p className="text-sm leading-relaxed text-ink-muted">{body}</p>
        {state !== "active" && (
          <Link
            href="/"
            className="mt-6 inline-block text-xs text-accent hover:underline"
          >
            Back to setup
          </Link>
        )}
      </div>
    </CenteredCard>
  );
}
