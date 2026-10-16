import { ReactNode } from "react";

export function CenteredCard({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-page px-6 py-12">
      <div className="w-full max-w-[440px] rounded-[20px] bg-surface p-11 shadow-[0_1px_2px_rgba(16,24,40,0.04),0_16px_40px_-12px_rgba(16,24,40,0.14)]">
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
