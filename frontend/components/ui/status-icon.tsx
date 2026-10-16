export function MailIcon() {
  return (
    <div className="mx-auto mb-6 flex h-12 w-12 items-center justify-center rounded-full bg-accent-soft">
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M3 7L12 14L21 7"
          stroke="var(--color-accent)"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <rect x="3" y="5" width="18" height="14" rx="2" stroke="var(--color-accent)" strokeWidth="2" />
      </svg>
    </div>
  );
}

export function CheckIcon() {
  return (
    <div className="mx-auto mb-6 flex h-12 w-12 items-center justify-center rounded-full bg-success-soft">
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M5 13L10 18L19 7"
          stroke="var(--color-success)"
          strokeWidth="2.4"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </div>
  );
}
