import { ReactNode, CSSProperties } from "react";

// Overrides the accent palette for everything inside it via CSS custom
// properties - every bg-accent/text-accent/border-accent (and the
// -secondary/-tertiary variants) utility in this project compiles to
// var(--color-accent...) under the hood (Tailwind v4's theme-variable
// architecture), so this is a real cascade override, not a style
// re-implementation. No colors at all means no wrapper - zero cost,
// identical to not having this component in the tree. Each of the three
// shades overrides independently, so setting only a primary still works.
export function ThemeScope({
  accentColor,
  accentColorSecondary,
  accentColorTertiary,
  children,
}: {
  accentColor?: string | null;
  accentColorSecondary?: string | null;
  accentColorTertiary?: string | null;
  children: ReactNode;
}) {
  if (!accentColor && !accentColorSecondary && !accentColorTertiary) {
    return <>{children}</>;
  }

  const style: CSSProperties = {};
  if (accentColor) {
    Object.assign(style, {
      "--color-accent": accentColor,
      "--color-accent-soft": `${accentColor}24`,
    });
  }
  if (accentColorSecondary) {
    Object.assign(style, {
      "--color-accent-secondary": accentColorSecondary,
      "--color-accent-secondary-soft": `${accentColorSecondary}24`,
    });
  }
  if (accentColorTertiary) {
    Object.assign(style, {
      "--color-accent-tertiary": accentColorTertiary,
      "--color-accent-tertiary-soft": `${accentColorTertiary}24`,
    });
  }

  return <div style={style}>{children}</div>;
}
