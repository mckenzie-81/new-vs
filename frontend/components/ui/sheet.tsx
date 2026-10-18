"use client";

export function Sheet({
  open,
  onClose,
  side = "right",
  title,
  widthClassName = "w-80 max-w-[85vw]",
  className = "",
  children,
}: {
  open: boolean;
  onClose: () => void;
  side?: "left" | "right";
  title?: string;
  widthClassName?: string;
  className?: string;
  children: React.ReactNode;
}) {
  if (!open) return null;

  const sideClasses =
    side === "right" ? "right-0 border-l border-line" : "left-0 border-r border-line";
  const slideInClass = side === "right" ? "sheet-slide-in-right" : "sheet-slide-in-left";

  return (
    <div className={`fixed inset-0 z-40 ${className}`}>
      <button
        aria-label="Close"
        className="overlay-fade-in absolute inset-0 bg-black/30"
        onClick={onClose}
      />
      <div
        className={`${slideInClass} absolute top-0 flex h-full flex-col bg-surface ${widthClassName} ${sideClasses}`}
      >
        {title && (
          <div className="flex items-center justify-between p-5">
            <h2 className="text-base font-semibold text-ink">{title}</h2>
            <button
              onClick={onClose}
              className="text-sm font-semibold text-ink-muted hover:text-ink"
            >
              Close
            </button>
          </div>
        )}
        {children}
      </div>
    </div>
  );
}
