import { AdminHeader } from "./admin-header";
import { Breadcrumbs } from "./breadcrumbs";

export function StubPage({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div>
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: title }]} />
      <AdminHeader title={title} />
      <p className="max-w-2xl text-sm text-ink-muted">{description}</p>
      <p className="py-10 text-sm text-ink-faint">Not built yet.</p>
    </div>
  );
}
