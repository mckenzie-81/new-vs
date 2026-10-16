import { ButtonHTMLAttributes, forwardRef } from "react";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className = "", ...props },
  ref
) {
  return (
    <button
      ref={ref}
      className={`w-full rounded-[10px] bg-ink px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-black disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
      {...props}
    />
  );
});
