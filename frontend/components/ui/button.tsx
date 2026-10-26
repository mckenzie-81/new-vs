import { ButtonHTMLAttributes, forwardRef } from "react";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "accent" | "danger" | "success";
  fullWidth?: boolean;
};

// Exactly one bg-* class applies at a time, picked here rather than left to
// whatever a caller appends via className - two conflicting bg-* utilities
// in one string aren't reliably resolved by source order, only by
// Tailwind's own internal generation order. The same reasoning is why
// width is a prop (fullWidth) instead of a className override: w-full and
// w-auto have the same footgun, and a screenshot once proved it.
//
// default is the brand accent color, not a neutral black - the primary
// action on a screen should read as branded by default, same as "accent"
// (kept as an explicit alias for call sites that already name it). A
// flat black default was the old convention; this app's buttons are
// meant to look like an institution's own console, not a generic one.
const fillClasses: Record<NonNullable<ButtonProps["variant"]>, string> = {
  default: "bg-accent hover:bg-accent-secondary",
  accent: "bg-accent hover:bg-accent-secondary",
  danger: "bg-danger hover:bg-danger/90",
  success: "bg-success hover:bg-success/90",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className = "", variant = "default", fullWidth = true, ...props },
  ref
) {
  return (
    <button
      ref={ref}
      className={`inline-flex min-h-11 ${fullWidth ? "w-full" : ""} items-center justify-center ${fillClasses[variant]} px-5 text-sm font-semibold text-white transition-colors duration-150 disabled:opacity-60 disabled:cursor-not-allowed ${className}`}
      {...props}
    />
  );
});

type SecondaryButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: "default" | "danger";
};

const textClasses: Record<NonNullable<SecondaryButtonProps["tone"]>, string> = {
  // The secondary brand shade on hover - a lighter touch than Button's
  // solid accent fill, for the less-emphasized action next to it.
  default: "text-ink hover:text-accent-secondary",
  danger: "text-danger",
};

const borderHoverClasses: Record<NonNullable<SecondaryButtonProps["tone"]>, string> = {
  default: "hover:border-accent-secondary",
  danger: "hover:border-danger",
};

export const SecondaryButton = forwardRef<HTMLButtonElement, SecondaryButtonProps>(
  function SecondaryButton({ className = "", tone = "default", ...props }, ref) {
    return (
      <button
        ref={ref}
        className={`inline-flex min-h-11 items-center border border-line px-4 text-sm font-semibold ${textClasses[tone]} ${borderHoverClasses[tone]} disabled:opacity-60 disabled:cursor-not-allowed ${className}`}
        {...props}
      />
    );
  }
);
