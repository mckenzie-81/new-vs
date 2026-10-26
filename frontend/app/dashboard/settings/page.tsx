"use client";

import { useEffect, useState } from "react";
import { getBranding, updateBranding, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { BrandingFields } from "@/components/branding/branding-fields";

export default function SettingsPage() {
  const [platformName, setPlatformName] = useState("");
  const [accentColor, setAccentColor] = useState("");
  const [accentColorSecondary, setAccentColorSecondary] = useState("");
  const [accentColorTertiary, setAccentColorTertiary] = useState("");
  const [logoUrl, setLogoUrl] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    getBranding()
      .then((settings) => {
        setPlatformName(settings.platformName ?? "");
        setAccentColor(settings.accentColor ?? "");
        setAccentColorSecondary(settings.accentColorSecondary ?? "");
        setAccentColorTertiary(settings.accentColorTertiary ?? "");
        setLogoUrl(settings.logoUrl ?? "");
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, []);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSaved(false);
    setSaving(true);
    try {
      const settings = await updateBranding(
        accentColor,
        accentColorSecondary,
        accentColorTertiary,
        logoUrl,
        platformName
      );
      setPlatformName(settings.platformName ?? "");
      setAccentColor(settings.accentColor ?? "");
      setAccentColorSecondary(settings.accentColorSecondary ?? "");
      setAccentColorTertiary(settings.accentColorTertiary ?? "");
      setLogoUrl(settings.logoUrl ?? "");
      setSaved(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to save.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="max-w-md">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-faint">Branding</h2>
      <p className="mt-1 text-sm text-ink-muted">
        The default look for the dashboard and for any election that hasn&apos;t set its own
        branding. Leave blank to use the built-in defaults.
      </p>

      {!loaded ? (
        <p className="py-10 text-sm text-ink-faint">Loading...</p>
      ) : (
        <form onSubmit={handleSave} className="mt-5 flex flex-col gap-4">
          <Field
            label="Platform name"
            htmlFor="platform-name"
            hint="Shown at the top of the dashboard sidebar. Leave blank for the default, &quot;Voting System&quot;."
          >
            <Input
              id="platform-name"
              placeholder="Voting System"
              value={platformName}
              onChange={(e) => setPlatformName(e.target.value)}
            />
          </Field>

          <BrandingFields
            accentColor={accentColor}
            accentColorSecondary={accentColorSecondary}
            accentColorTertiary={accentColorTertiary}
            logoUrl={logoUrl}
            onAccentColorChange={setAccentColor}
            onAccentColorSecondaryChange={setAccentColorSecondary}
            onAccentColorTertiaryChange={setAccentColorTertiary}
            onLogoUrlChange={setLogoUrl}
          />

          {error && <p className="text-sm text-danger">{error}</p>}
          {saved && !error && <p className="text-sm text-success">Saved.</p>}

          <Button type="submit" variant="accent" disabled={saving} className="mt-1" fullWidth={false}>
            {saving ? "Saving..." : "Save"}
          </Button>
        </form>
      )}
    </div>
  );
}
