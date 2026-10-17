const tone: Record<string, string> = {
  ACTIVE: "bg-success text-white",
  PENDING_VERIFICATION: "bg-warning text-white",
  DISABLED: "bg-danger text-white",
};

const label: Record<string, string> = {
  ACTIVE: "active",
  PENDING_VERIFICATION: "pending",
  DISABLED: "disabled",
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
