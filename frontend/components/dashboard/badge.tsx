const tone: Record<string, string> = {
  // People
  ACTIVE: "bg-success text-white",
  PENDING_VERIFICATION: "bg-warning text-white",
  DISABLED: "bg-danger text-white",
  // Elections
  DRAFT: "bg-page text-ink-muted",
  PENDING_APPROVAL: "bg-warning text-white",
  APPROVED: "bg-accent text-white",
  LIVE: "bg-success text-white",
  CLOSED: "bg-ink text-white",
  ARCHIVED: "bg-page text-ink-faint",
};

const label: Record<string, string> = {
  ACTIVE: "active",
  PENDING_VERIFICATION: "pending",
  DISABLED: "disabled",
  DRAFT: "draft",
  PENDING_APPROVAL: "pending approval",
  APPROVED: "approved",
  LIVE: "live",
  CLOSED: "closed",
  ARCHIVED: "archived",
};

export function Badge({ status }: { status: string }) {
  return (
    <span
      className={`inline-block px-2 py-0.5 text-xs font-semibold capitalize ${
        tone[status] ?? "bg-page text-ink-muted"
      }`}
    >
      {label[status] ?? status}
    </span>
  );
}
