import { HTMLAttributes, forwardRef } from "react";

// A tint-block container, not a bordered box with a shadow - square
// corners, a single hairline border in --color-line, nothing else. The
// one shared wrapper behind stat tiles and the search/filter+table
// pattern used across the list pages.
export const Card = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  function Card({ className = "", ...props }, ref) {
    return <div ref={ref} className={`border border-line bg-surface ${className}`} {...props} />;
  }
);

export const CardHeader = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  function CardHeader({ className = "", ...props }, ref) {
    return <div ref={ref} className={`flex items-center justify-between gap-2 p-5 pb-2 ${className}`} {...props} />;
  }
);

export const CardContent = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  function CardContent({ className = "", ...props }, ref) {
    return <div ref={ref} className={`p-5 pt-0 ${className}`} {...props} />;
  }
);
