"use client";

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        aria-label="Close"
        className="absolute inset-0 bg-black/30"
        onClick={onClose}
      />
      <div className="relative w-full max-w-sm border border-line bg-surface p-6">
        <div className="mb-5 flex items-center justify-between gap-4">
          <h2 className="text-base font-semibold text-ink">{title}</h2>
          <button
            onClick={onClose}
            className="text-sm font-semibold text-ink-muted hover:text-ink"
          >
            Close
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
