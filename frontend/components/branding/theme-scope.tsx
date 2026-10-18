import { ReactNode, CSSProperties } from "react";

// Overrides the accent color for everything inside it via CSS custom
// properties - every bg-accent/text-accent/border-accent utility in this
// project compiles to var(--color-accent) under the hood (Tailwind v4's
// theme-variable architecture), so this is a real cascade override, not a
// style re-implementation. No accentColor means no wrapper at all - zero
// cost, identical to not having this component in the tree.
export function ThemeScope({
  accentColor,
  children,
}: {
  accentColor?: string | null;
  children: ReactNode;
}) {
  if (!accentColor) return <>{children}</>;

  const style = {
    "--color-accent": accentColor,
    "--color-accent-soft": `${accentColor}24`,
  } as CSSProperties;

  return <div style={style}>{children}</div>;
}
