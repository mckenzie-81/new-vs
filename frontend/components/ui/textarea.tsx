import { TextareaHTMLAttributes, forwardRef } from "react";

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className = "", rows = 3, ...props },
  ref
) {
  return (
    <textarea
      ref={ref}
      rows={rows}
      className={`mt-2 block w-full resize-none border-0 border-b border-line bg-transparent px-0 py-2 text-base text-ink outline-none placeholder:text-ink-faint focus:border-ink aria-[invalid=true]:border-danger ${className}`}
      {...props}
    />
  );
});
