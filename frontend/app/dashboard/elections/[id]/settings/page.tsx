"use client";

import { useState } from "react";
import {
  updateElection,
  electionToCreateInput,
  ApiError,
  BALLOT_STYLES,
} from "@/lib/api";
import { useDashboardUser } from "@/components/dashboard/user-context";
import { useElection } from "@/components/dashboard/election-context";
import { AdminHeader } from "@/components/dashboard/admin-header";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Field } from "@/components/ui/field";
import { BrandingFields } from "@/components/branding/branding-fields";
import { BallotStylePicker } from "@/components/ballot/ballot-style-picker";

export default function ElectionSettingsPage() {
  const user = useDashboardUser();
  const { election, assignedAdmins, refresh } = useElection();

  const canEdit =
    election.status === "DRAFT" &&
    (user.role === "SYSTEMADMIN" ||
      (user.role === "ADMIN" && assignedAdmins.some((a) => a.userId === user.id)));

  const [description, setDescription] = useState(election.description ?? "");
  const [accentColor, setAccentColor] = useState(election.accentColor ?? "");
  const [logoUrl, setLogoUrl] = useState(election.logoUrl ?? "");
  const [ballotStyle, setBallotStyle] = useState(election.ballotStyle);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSaved(false);
    setSubmitting(true);
    try {
      await updateElection(election.id, {
        ...electionToCreateInput(election),
        description,
        accentColor,
        logoUrl,
        ballotStyle,
      });
      setSaved(true);
      refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }

  const ballotStyleLabel =
    BALLOT_STYLES.find((s) => s.value === election.ballotStyle)?.label ?? election.ballotStyle;

  if (!canEdit) {
    return (
      <div>
        <AdminHeader
          title="Settings"
          intro={
            election.status === "DRAFT"
              ? "Only a systemadmin or an assigned admin can change these."
              : "Settings can only be changed while this election is a draft."
          }
        />
        <div className="flex flex-col gap-6 border-t border-line pt-6">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
              Description
            </p>
            <p className="mt-1 text-sm text-ink-muted">
              {election.description || "None."}
            </p>
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
              Branding
            </p>
            <p className="mt-1 text-sm text-ink-muted">
              {election.accentColor || election.logoUrl
                ? "Custom branding set."
                : "Using the system default."}
            </p>
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-faint">
              Ballot style
            </p>
            <p className="mt-1 text-sm text-ink-muted">{ballotStyleLabel}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div>
      <AdminHeader
        title="Settings"
        intro="Description, branding, and ballot layout for this election."
      />

      <form onSubmit={handleSubmit} className="flex flex-col gap-6 border-t border-line pt-6">
        <Field label="Description" htmlFor="election-description" hint="Optional">
          <Textarea
            id="election-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>

        <div>
          <p className="text-sm font-semibold text-ink">Branding</p>
          <p className="mt-1 text-sm text-ink-muted">
            Optional - leave blank to use the system default.
          </p>
          <div className="mt-3">
            <BrandingFields
              accentColor={accentColor}
              logoUrl={logoUrl}
              onAccentColorChange={setAccentColor}
              onLogoUrlChange={setLogoUrl}
            />
          </div>
          {(accentColor || logoUrl) && (
            <button
              type="button"
              onClick={() => {
                setAccentColor("");
                setLogoUrl("");
              }}
              className="mt-2 text-sm font-semibold text-accent hover:underline"
            >
              Use system default
            </button>
          )}
        </div>

        <div>
          <p className="text-sm font-semibold text-ink">Ballot style</p>
          <p className="mt-1 text-sm text-ink-muted">How candidates are laid out on the ballot.</p>
          <div className="mt-3">
            <BallotStylePicker value={ballotStyle} onChange={setBallotStyle} />
          </div>
        </div>

        {error && <p className="text-sm text-danger">{error}</p>}
        {saved && !error && <p className="text-sm text-success">Saved.</p>}

        <Button type="submit" disabled={submitting} className="mt-1" fullWidth={false}>
          {submitting ? "Saving..." : "Save"}
        </Button>
      </form>
    </div>
  );
}
