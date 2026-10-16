import { ButtonHTMLAttributes, forwardRef } from "react";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className = "", ...props },
  ref
) {
  return (
    <button
      ref={ref}
      className={`inline-flex min-h-11 w-full items-center justify-center bg-ink px-5 text-sm font-semibold text-white disabled:opacity-60 disabled:cursor-not-allowed ${className}`}
      {...props}
    />
  );
});
