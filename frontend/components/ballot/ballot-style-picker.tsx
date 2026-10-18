"use client";

import { BALLOT_STYLES } from "@/lib/api";

export function BallotStylePicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      {BALLOT_STYLES.map((option) => {
        const selected = value === option.value;
        return (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={`flex items-start gap-3 border-2 p-3 text-left transition-colors duration-150 ${
              selected ? "border-accent" : "border-line hover:border-ink"
            }`}
          >
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
          </button>
        );
      })}
    </div>
  );
}
