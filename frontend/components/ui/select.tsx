import { SelectHTMLAttributes, forwardRef } from "react";

type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className = "", ...props },
  ref
) {
  return (
    <select
      ref={ref}
      className={`mt-2 block min-h-11 w-full border-0 border-b border-line bg-transparent px-0 py-2 text-base text-ink outline-none focus:border-ink ${className}`}
      {...props}
    />
  );
});
