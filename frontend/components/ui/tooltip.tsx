export function Tooltip({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <span className="group relative inline-flex">
      {children}
      <span
        role="tooltip"
        className="pointer-events-none absolute -top-9 left-1/2 z-10 -translate-x-1/2 whitespace-nowrap bg-ink px-2 py-1 text-xs font-medium text-white opacity-0 transition-opacity delay-150 group-hover:opacity-100"
      >
        {label}
      </span>
    </span>
  );
}
