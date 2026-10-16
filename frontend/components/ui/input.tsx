import { InputHTMLAttributes, forwardRef } from "react";

type InputProps = InputHTMLAttributes<HTMLInputElement>;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className = "", ...props },
  ref
) {
  return (
    <input
      ref={ref}
      className={`mt-2 block min-h-11 w-full border-0 border-b border-line bg-transparent px-0 py-2 text-base text-ink outline-none placeholder:text-ink-faint focus:border-ink aria-[invalid=true]:border-danger ${className}`}
      {...props}
    />
  );
});
