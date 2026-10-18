"use client";

import { CheckIcon } from "@heroicons/react/24/solid";
import { Candidate, candidateTicketName } from "@/lib/api";

// Shared between the real voter ballot and the admin's "voter preview" -
// one set of layouts, not copies that could quietly drift apart.
//
// Selection always uses the accent color, not green, in both layouts -
// green is already this app's "success/done" color (already-voted,
// completed actions), and reusing it here would make an in-progress pick
// look like a confirmed outcome before it is one.
function SelectionBadge({ selected, size }: { selected: boolean; size: "sm" | "md" }) {
  const dims = size === "sm" ? "h-5 w-5" : "h-6 w-6";
  const icon = size === "sm" ? "h-3 w-3" : "h-3.5 w-3.5";
  return (
    <span
      aria-hidden
      className={`flex shrink-0 items-center justify-center rounded-full border-2 transition-colors duration-150 ${dims} ${
        selected ? "border-accent bg-accent" : "border-line bg-surface"
      }`}
    >
      {selected && <CheckIcon className={`${icon} text-white`} />}
    </span>
  );
}

function CandidatePhoto({
  candidate,
  className,
}: {
  candidate: Candidate;
  className: string;
}) {
  if (candidate.photoUrl) {
    // next/image needs a fixed set of allowed hosts; the host here isn't
    // decided yet (Cloudinary vs. our own backend), so a plain img tag
    // avoids hardcoding one prematurely.
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={candidate.photoUrl} alt="" className={`${className} bg-page object-cover`} />;
  }
  return (
    <div className={`${className} flex items-center justify-center bg-page font-semibold text-ink-faint`}>
      {candidate.fullName.trim().charAt(0).toUpperCase() || "?"}
    </div>
  );
}

// "compact": a dense row - small photo, name beside it. Good for long
// candidate lists where scanning quickly matters more than a large photo.
function CompactCandidateOption({
  candidate,
  selected,
  onSelect,
}: {
  candidate: Candidate;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`relative flex w-full items-center gap-3 border-2 p-2.5 text-left transition-colors duration-150 ${
        selected ? "border-accent" : "border-line hover:border-ink"
      }`}
    >
      <SelectionBadge selected={selected} size="sm" />
      <CandidatePhoto candidate={candidate} className="h-16 w-14 shrink-0 text-xl" />
      <span className="text-base font-semibold leading-snug text-ink">
        {candidateTicketName(candidate)}
      </span>
    </button>
  );
}

// "cards": a two-column grid of taller cards - a bigger photo matters more
// here, bounded by the grid cell rather than the full ballot width, so it
// doesn't dominate the screen across several candidates and portfolios.
function CardCandidateOption({
  candidate,
  selected,
  onSelect,
}: {
  candidate: Candidate;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`relative w-full border-2 p-2.5 text-center transition-colors duration-150 ${
        selected ? "border-accent" : "border-line hover:border-ink"
      }`}
    >
      <span className="absolute left-2 top-2 z-10">
        <SelectionBadge selected={selected} size="md" />
      </span>
      <CandidatePhoto candidate={candidate} className="aspect-[3/4] w-full text-3xl" />
      <span className="mt-2 block text-sm font-semibold leading-snug text-ink">
        {candidateTicketName(candidate)}
      </span>
    </button>
  );
}

export function CandidateOption({
  candidate,
  selected,
  onSelect,
  style,
}: {
  candidate: Candidate;
  selected: boolean;
  onSelect: () => void;
  style: string;
}) {
  if (style === "cards") {
    return <CardCandidateOption candidate={candidate} selected={selected} onSelect={onSelect} />;
  }
  return <CompactCandidateOption candidate={candidate} selected={selected} onSelect={onSelect} />;
}

// The container each style's candidates sit in - compact is a single
// stacked column, cards is a two-column grid so the bigger photo stays
// bounded to roughly half the ballot's width instead of stretching full
// width.
export function candidateListClassName(style: string) {
  return style === "cards" ? "grid grid-cols-2 gap-3" : "flex flex-col gap-3";
}
