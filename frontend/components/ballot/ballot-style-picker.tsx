"use client";

import { BALLOT_STYLES } from "@/lib/api";

// Small wireframe mockups, not real candidate data - the point is to show
// the shape of each layout (a compact row vs. a taller card, where the
// photo sits, how big it is) at a glance, not to pixel-match the real
// ballot. Abstract and static means this can't drift out of sync with
// real candidate data, and stays pure decoration (aria-hidden) rather
// than a second, confusing set of interactive-looking elements nested
// inside this picker's own button.
function CompactPreview() {
  return (
    <div className="flex flex-col gap-1.5" aria-hidden>
      {[0, 1].map((i) => (
        <div key={i} className="flex items-center gap-1.5 border border-line p-1.5">
          <span className="h-3 w-3 shrink-0 rounded-full border border-line" />
          <span className="h-6 w-5 shrink-0 bg-line" />
          <span className="h-1.5 flex-1 bg-line" />
        </div>
      ))}
    </div>
  );
}

function CardsPreview() {
  return (
    <div className="flex flex-col gap-1.5" aria-hidden>
      {[0, 1].map((i) => (
        <div key={i} className="relative border border-line p-1.5">
          <span className="absolute left-1 top-1 h-2.5 w-2.5 rounded-full border border-line bg-surface" />
          <span className="block aspect-[3/4] w-full bg-line" />
          <span className="mt-1.5 block h-1.5 w-3/4 bg-line" />
        </div>
      ))}
    </div>
  );
}

const PREVIEWS: Record<string, () => React.JSX.Element> = {
  compact: CompactPreview,
  cards: CardsPreview,
};

export function BallotStylePicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-3">
      {BALLOT_STYLES.map((option) => {
        const selected = value === option.value;
        const Preview = PREVIEWS[option.value];
        return (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={`flex flex-col gap-3 border-2 p-3 text-left transition-colors duration-150 ${
              selected ? "border-accent" : "border-line hover:border-ink"
            }`}
          >
            {Preview && (
              <div className="bg-page p-2">
                <Preview />
              </div>
            )}
            <div className="flex items-start gap-2.5">
              <span
                aria-hidden
                className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 ${
                  selected ? "border-accent" : "border-line"
                }`}
              >
                {selected && <span className="h-2.5 w-2.5 rounded-full bg-accent" />}
              </span>
              <span>
                <span className="block text-sm font-semibold text-ink">{option.label}</span>
                <span className="mt-0.5 block text-sm text-ink-muted">{option.description}</span>
              </span>
            </div>
          </button>
        );
      })}
    </div>
  );
}
