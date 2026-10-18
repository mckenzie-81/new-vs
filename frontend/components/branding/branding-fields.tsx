"use client";

import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";

const FALLBACK_SWATCH = "#2e4e8f";

export function BrandingFields({
  accentColor,
  logoUrl,
  onAccentColorChange,
  onLogoUrlChange,
  accentColorHint,
}: {
  accentColor: string;
  logoUrl: string;
  onAccentColorChange: (value: string) => void;
  onLogoUrlChange: (value: string) => void;
  accentColorHint?: string;
}) {
  return (
    <div className="flex flex-col gap-4">
      <Field label="Accent color" htmlFor="brand-accent-color" hint={accentColorHint}>
        <div className="flex items-center gap-3">
          <input
            type="color"
            aria-label="Pick accent color"
            value={/^#[0-9a-fA-F]{6}$/.test(accentColor) ? accentColor : FALLBACK_SWATCH}
            onChange={(e) => onAccentColorChange(e.target.value)}
            className="h-11 w-11 shrink-0 cursor-pointer border border-line bg-transparent p-1"
          />
          <Input
            id="brand-accent-color"
            placeholder={FALLBACK_SWATCH}
            value={accentColor}
            onChange={(e) => onAccentColorChange(e.target.value)}
          />
        </div>
      </Field>

      <Field
        label="Logo URL"
        htmlFor="brand-logo-url"
        hint="Shown on the ballot and the public pulse page."
      >
        <Input
          id="brand-logo-url"
          placeholder="https://..."
          value={logoUrl}
          onChange={(e) => onLogoUrlChange(e.target.value)}
        />
      </Field>
    </div>
  );
}
