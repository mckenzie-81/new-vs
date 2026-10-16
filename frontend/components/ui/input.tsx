import { InputHTMLAttributes, forwardRef } from "react";

type InputProps = InputHTMLAttributes<HTMLInputElement>;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className = "", ...props },
  ref
) {
  return (
    <input
      ref={ref}
      className={`w-full rounded-[10px] border border-line bg-surface px-[14px] py-[11px] text-sm text-ink outline-none transition-colors focus:border-accent focus:ring-[3px] focus:ring-accent-soft ${className}`}
      {...props}
    />
  );
});
