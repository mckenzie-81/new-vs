"use client";

import { Input } from "@/components/ui/input";
import { Field } from "@/components/ui/field";
import { ImageUpload } from "@/components/ui/image-upload";

const FALLBACK_SWATCHES = ["#2f4a75", "#243a5c", "#b79a64"];

function ColorField({
  id,
  label,
  hint,
  value,
  fallback,
  onChange,
}: {
  id: string;
  label: string;
  hint?: string;
  value: string;
  fallback: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field label={label} htmlFor={id} hint={hint}>
      <div className="flex items-center gap-3">
        <input
          type="color"
          aria-label={`Pick ${label.toLowerCase()}`}
          value={/^#[0-9a-fA-F]{6}$/.test(value) ? value : fallback}
          onChange={(e) => onChange(e.target.value)}
          className="h-11 w-11 shrink-0 cursor-pointer border border-line bg-transparent p-1"
        />
        <Input id={id} placeholder={fallback} value={value} onChange={(e) => onChange(e.target.value)} />
      </div>
    </Field>
  );
}

// Three shades, not one - a primary (the main brand color, used for
// buttons/selection same as before), a secondary (a second brand color
// for secondary actions and accents), and a tertiary (the subtlest, for
// small decorative touches). Still accent-only: none of this touches the
// neutral text/background palette or the fixed success/warning/danger
// colors, same guarantee as a single accent color always had.
export function BrandingFields({
  accentColor,
  accentColorSecondary,
  accentColorTertiary,
  logoUrl,
  onAccentColorChange,
  onAccentColorSecondaryChange,
  onAccentColorTertiaryChange,
  onLogoUrlChange,
  accentColorHint,
}: {
  accentColor: string;
  accentColorSecondary: string;
  accentColorTertiary: string;
  logoUrl: string;
  onAccentColorChange: (value: string) => void;
  onAccentColorSecondaryChange: (value: string) => void;
  onAccentColorTertiaryChange: (value: string) => void;
  onLogoUrlChange: (value: string) => void;
  accentColorHint?: string;
}) {
  return (
    <div className="flex flex-col gap-4">
      <ColorField
        id="brand-accent-color"
        label="Primary color"
        hint={accentColorHint}
        value={accentColor}
        fallback={FALLBACK_SWATCHES[0]}
        onChange={onAccentColorChange}
      />
      <ColorField
        id="brand-accent-color-secondary"
        label="Secondary color"
        hint="Optional."
        value={accentColorSecondary}
        fallback={FALLBACK_SWATCHES[1]}
        onChange={onAccentColorSecondaryChange}
      />
      <ColorField
        id="brand-accent-color-tertiary"
        label="Tertiary color"
        hint="Optional."
        value={accentColorTertiary}
        fallback={FALLBACK_SWATCHES[2]}
        onChange={onAccentColorTertiaryChange}
      />

      <Field
        label="Logo"
        htmlFor="brand-logo-url"
        hint="Shown on the ballot and the public pulse page."
      >
        <div className="mt-1">
          <ImageUpload id="brand-logo-url" kind="logos" value={logoUrl} onChange={onLogoUrlChange} />
        </div>
      </Field>
    </div>
  );
}
