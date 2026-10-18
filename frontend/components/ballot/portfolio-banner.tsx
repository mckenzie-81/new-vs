// Shared between the real voter ballot and the admin's "voter preview" -
// a colored strip per portfolio so a long ballot with several portfolios
// (President, VP, Secretary...) reads as distinct sections at a glance,
// not one long undifferentiated list.
export function PortfolioBanner({
  title,
  onReset,
}: {
  title: string;
  onReset?: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-3 border-l-2 border-accent bg-accent-soft py-2.5 pl-3 pr-3">
      <h3 className="text-base font-semibold tracking-wide text-ink">{title}</h3>
      {onReset && (
        <button
          type="button"
          onClick={onReset}
          className="shrink-0 text-sm font-semibold text-ink-faint transition-colors duration-150 hover:text-danger"
        >
          Reset
        </button>
      )}
    </div>
  );
}
