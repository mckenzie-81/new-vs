import { ReactNode } from "react";

export function CenteredCard({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-page px-6 py-12">
      <div className="w-full max-w-[420px] border border-line bg-surface p-10">
        {children}
      </div>
    </div>
  );
}

export function CenteredMessage({ children }: { children: ReactNode }) {
  return (
    <CenteredCard>
      <p className="text-center text-sm leading-relaxed text-ink-muted">{children}</p>
    </CenteredCard>
  );
}
