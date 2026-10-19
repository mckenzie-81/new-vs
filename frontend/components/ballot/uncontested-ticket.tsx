import { CheckIcon } from "@heroicons/react/24/solid";
import { Candidate, candidateTicketName } from "@/lib/api";
import { CandidatePhoto } from "@/components/ballot/candidate-option";
import { Button } from "@/components/ui/button";

// An uncontested portfolio (exactly one candidate/ticket) isn't a choice
// between options - there's nothing to compare it against, so a normal
// selectable card (or a grid of one) doesn't make sense. This asks the
// real question instead: approve this candidate/ticket, or not. "No" is
// a deliberate, tracked rejection (see is_explicit_no on the backend),
// not the same thing as just not answering.
export function UncontestedTicket({
  candidate,
  portfolioTitle,
  response,
  onRespond,
}: {
  candidate: Candidate;
  portfolioTitle: string;
  response: "yes" | "no" | null;
  onRespond: (response: "yes" | "no" | null) => void;
}) {
  const borderClass =
    response === "yes" ? "border-success" : response === "no" ? "border-danger" : "border-line";

  return (
    <div className={`relative border-2 p-4 text-center transition-colors duration-150 ${borderClass}`}>
      {response === "yes" && (
        <span className="absolute left-3 top-3 flex h-7 w-7 items-center justify-center rounded-full bg-success">
          <CheckIcon className="h-4 w-4 text-white" />
        </span>
      )}
      <CandidatePhoto candidate={candidate} className="mx-auto aspect-[4/5] w-full max-w-xs text-4xl" />
      <p className="mt-3 text-xl font-semibold leading-snug text-ink">
        {candidateTicketName(candidate)}
      </p>
      <p className="mt-1 text-sm text-ink-muted">As {portfolioTitle}</p>

      <div className="mt-4 flex flex-col gap-2.5">
        <Button
          type="button"
          variant="success"
          onClick={() => onRespond(response === "yes" ? null : "yes")}
        >
          Yes
        </Button>
        <Button
          type="button"
          variant="danger"
          onClick={() => onRespond(response === "no" ? null : "no")}
        >
          No
        </Button>
      </div>
    </div>
  );
}
