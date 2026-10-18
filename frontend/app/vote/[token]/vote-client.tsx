"use client";

import { useEffect, useState } from "react";
import {
  getVoteTokenStatus,
  authenticateVoter,
  getVoterBallot,
  castVote,
  ApiError,
  VoteElectionSummary,
  Portfolio,
  Candidate,
} from "@/lib/api";
import { CenteredCard, CenteredMessage } from "@/components/layout/centered-card";
import { Button, SecondaryButton } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  ClockIcon,
} from "@heroicons/react/24/outline";

type Screen =
  | "loading"
  | "invalid"
  | "locked"
  | "already_voted"
  | "not_open"
  | "verify"
  | "ballot"
  | "review"
  | "done"
  | "error";

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

// portfolioId -> candidateId; an absent key means abstain, same convention
// as the admin-side VoterPreviewModal this screen is adapted from.
type Selections = Record<string, string>;

function CandidateOption({
  number,
  candidate,
  selected,
  onSelect,
}: {
  number: number;
  candidate: Candidate;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`flex w-full items-center gap-4 border p-3 text-left ${
        selected ? "border-accent bg-accent-soft" : "border-line hover:border-ink"
      }`}
    >
      <span className="w-6 shrink-0 text-center text-base font-semibold text-ink-faint">
        {number}
      </span>
      {candidate.photoUrl ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={candidate.photoUrl}
          alt=""
          className="h-20 w-20 shrink-0 bg-page object-cover"
        />
      ) : (
        <div className="flex h-20 w-20 shrink-0 items-center justify-center bg-page text-2xl font-semibold text-ink-faint">
          {candidate.fullName.trim().charAt(0).toUpperCase() || "?"}
        </div>
      )}
      <span className="text-lg font-medium text-ink">{candidate.fullName}</span>
    </button>
  );
}

export function VoteClient({ token }: { token: string }) {
  const [screen, setScreen] = useState<Screen>("loading");
  const [election, setElection] = useState<VoteElectionSummary | null>(null);

  const [verificationColumns, setVerificationColumns] = useState<string[]>([]);
  const [verificationValues, setVerificationValues] = useState<Record<string, string>>({});
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);

  const [portfolios, setPortfolios] = useState<Portfolio[] | null>(null);
  const [selections, setSelections] = useState<Selections>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function loadBallot(electionId: string) {
    getVoterBallot(electionId)
      .then((ballot) => {
        setElection(ballot.election);
        setPortfolios(ballot.portfolios);
        setScreen("ballot");
      })
      .catch((err) => {
        if (err instanceof ApiError && err.status === 409) {
          setScreen("already_voted");
        } else if (err instanceof ApiError && err.status === 403) {
          setScreen("not_open");
        } else {
          setScreen("error");
        }
      });
  }

  useEffect(() => {
    getVoteTokenStatus(token)
      .then((status) => {
        setElection(status.election ?? null);
        switch (status.status) {
          case "invalid":
            setScreen("invalid");
            return;
          case "locked":
            setScreen("locked");
            return;
          case "voted":
            setScreen("already_voted");
            return;
          case "authenticated":
            if (status.election) loadBallot(status.election.id);
            return;
          default:
            setVerificationColumns(status.verificationColumns ?? []);
            setScreen("verify");
        }
      })
      .catch(() => setScreen("error"));
  }, [token]);

  async function handleVerify(e: React.FormEvent) {
    e.preventDefault();
    setVerifyError(null);
    setVerifying(true);
    try {
      await authenticateVoter(token, verificationValues);
      if (election) loadBallot(election.id);
    } catch (err) {
      if (err instanceof ApiError && err.status === 423) {
        setScreen("locked");
      } else if (err instanceof ApiError && err.status === 409) {
        setScreen("already_voted");
      } else {
        setVerifyError(
          err instanceof ApiError ? err.message : "Something went wrong. Please try again."
        );
      }
    } finally {
      setVerifying(false);
    }
  }

  // Selecting the already-selected candidate again clears it - abstain
  // isn't a choice you make, it's just what no selection means.
  function choose(portfolioId: string, candidateId: string) {
    setSelections((prev) => {
      const next = { ...prev };
      if (next[portfolioId] === candidateId) {
        delete next[portfolioId];
      } else {
        next[portfolioId] = candidateId;
      }
      return next;
    });
  }

  async function handleCast() {
    if (!election) return;
    setSubmitError(null);
    setSubmitting(true);
    try {
      await castVote(election.id, selections);
      setScreen("done");
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setScreen("already_voted");
      } else if (err instanceof ApiError && err.status === 403) {
        setScreen("not_open");
      } else {
        setSubmitError(
          err instanceof ApiError ? err.message : "Something went wrong. Please try again."
        );
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (screen === "loading") {
    return <CenteredMessage>Loading your voting link...</CenteredMessage>;
  }

  if (screen === "invalid") {
    return (
      <CenteredCard>
        <div className="text-center">
          <ExclamationTriangleIcon className="mx-auto mb-3 h-10 w-10 text-danger" />
          <p className="text-sm leading-relaxed text-ink-muted">
            This voting link isn&apos;t valid or has expired. Contact your election
            administrator if you think this is a mistake.
          </p>
        </div>
      </CenteredCard>
    );
  }

  if (screen === "locked") {
    return (
      <CenteredCard>
        <div className="text-center">
          <ExclamationTriangleIcon className="mx-auto mb-3 h-10 w-10 text-danger" />
          <p className="text-sm leading-relaxed text-ink-muted">
            This link has been locked after too many failed attempts. Contact your election
            administrator for a new one.
          </p>
        </div>
      </CenteredCard>
    );
  }

  if (screen === "error") {
    return <CenteredMessage>Something went wrong. Please try again later.</CenteredMessage>;
  }

  if (screen === "already_voted") {
    return (
      <CenteredCard>
        <div className="text-center">
          <CheckCircleIcon className="mx-auto mb-3 h-10 w-10 text-success" />
          <h1 className="mb-2 text-xl font-semibold tracking-tight">Already voted</h1>
          <p className="text-sm leading-relaxed text-ink-muted">
            You&apos;ve already cast your ballot{election ? ` for ${election.title}` : ""}.
            Thank you for voting.
          </p>
        </div>
      </CenteredCard>
    );
  }

  if (screen === "not_open") {
    return (
      <CenteredCard>
        <div className="text-center">
          <ClockIcon className="mx-auto mb-3 h-10 w-10 text-warning" />
          <h1 className="mb-2 text-xl font-semibold tracking-tight">Not open yet</h1>
          <p className="text-sm leading-relaxed text-ink-muted">
            {election
              ? `Voting for ${election.title} opens ${formatDateTime(election.startTime)}. Come back then using the same link.`
              : "Voting isn't open yet for this election. Come back and use the same link once it starts."}
          </p>
        </div>
      </CenteredCard>
    );
  }

  if (screen === "done") {
    return (
      <CenteredCard>
        <div className="text-center">
          <CheckCircleIcon className="mx-auto mb-3 h-10 w-10 text-success" />
          <h1 className="mb-2 text-xl font-semibold tracking-tight">Vote submitted</h1>
          <p className="text-sm leading-relaxed text-ink-muted">
            Thank you for voting{election ? ` in ${election.title}` : ""}. You can close this
            page now.
          </p>
        </div>
      </CenteredCard>
    );
  }

  if (screen === "verify") {
    return (
      <CenteredCard>
        <h1 className="mb-2 text-[23px] font-semibold leading-tight tracking-tight">
          {election?.title ?? "Vote"}
        </h1>
        <p className="mb-8 text-sm leading-relaxed text-ink-muted">
          {verificationColumns.length > 0
            ? "Confirm your details to continue to your ballot."
            : "Click below to continue to your ballot."}
        </p>
        <form onSubmit={handleVerify} className="flex flex-col gap-5">
          {verificationColumns.map((column) => (
            <Field key={column} label={column} htmlFor={`verify-${column}`}>
              <Input
                id={`verify-${column}`}
                required
                value={verificationValues[column] ?? ""}
                onChange={(e) =>
                  setVerificationValues((prev) => ({ ...prev, [column]: e.target.value }))
                }
              />
            </Field>
          ))}

          {verifyError && <p className="text-sm text-danger">{verifyError}</p>}

          <Button type="submit" disabled={verifying} className="mt-2">
            {verifying ? "Checking..." : "Continue"}
          </Button>
        </form>
      </CenteredCard>
    );
  }

  if (screen === "review") {
    return (
      <div className="mx-auto max-w-2xl px-6 py-10">
        <h1 className="text-2xl font-bold tracking-tight">Review your selections</h1>
        <ul className="mt-6 divide-y divide-line border-t border-line">
          {portfolios?.map((p) => {
            const candidate = p.candidates.find((c) => c.id === selections[p.id]);
            return (
              <li key={p.id} className="py-3">
                <div className="text-sm font-semibold text-ink">{p.title}</div>
                <div className="text-sm text-ink-muted">
                  {candidate ? candidate.fullName : "Abstained"}
                </div>
              </li>
            );
          })}
        </ul>

        {submitError && <p className="mt-4 text-sm text-danger">{submitError}</p>}

        <div className="mt-6 flex gap-3">
          <SecondaryButton onClick={() => setScreen("ballot")}>Back</SecondaryButton>
          <Button variant="accent" fullWidth={false} disabled={submitting} onClick={handleCast}>
            {submitting ? "Submitting..." : "Submit my vote"}
          </Button>
        </div>
      </div>
    );
  }

  // screen === "ballot"
  return (
    <div className="mx-auto max-w-2xl px-6 py-10">
      <h1 className="text-2xl font-bold tracking-tight">{election?.title}</h1>
      {election?.description && (
        <p className="mt-2 text-base text-ink-muted">{election.description}</p>
      )}

      {portfolios?.map((p) => (
        <div key={p.id} className="border-t border-line py-5 first:border-t-0">
          <h3 className="text-xl font-semibold text-ink">{p.title}</h3>
          <p className="mt-0.5 text-sm text-ink-faint">
            Tap a candidate to select them. Tap again, or leave blank, to abstain.
          </p>
          <div className="mt-4 flex flex-col gap-2">
            {p.candidates.map((c, index) => (
              <CandidateOption
                key={c.id}
                number={index + 1}
                candidate={c}
                selected={selections[p.id] === c.id}
                onSelect={() => choose(p.id, c.id)}
              />
            ))}
          </div>
        </div>
      ))}

      <div className="mt-5 flex gap-3">
        <SecondaryButton
          type="button"
          onClick={() => setSelections({})}
          disabled={Object.keys(selections).length === 0}
        >
          Reset selections
        </SecondaryButton>
        <Button variant="accent" fullWidth={false} onClick={() => setScreen("review")}>
          Review selections
        </Button>
      </div>
    </div>
  );
}
