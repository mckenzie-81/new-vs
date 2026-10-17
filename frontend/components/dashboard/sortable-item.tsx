"use client";

import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Bars2Icon } from "@heroicons/react/24/outline";

export function SortableItem({
  id,
  disabled,
  as: Tag = "div",
  data,
  children,
}: {
  id: string;
  disabled?: boolean;
  as?: "div" | "li";
  data?: Record<string, unknown>;
  children: React.ReactNode;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    disabled,
    data,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <Tag ref={setNodeRef} style={style} className="flex items-start gap-2">
      {!disabled && (
        <button
          {...attributes}
          {...listeners}
          aria-label="Drag to reorder"
          className="mt-1 shrink-0 cursor-grab touch-none text-ink-faint hover:text-ink-muted active:cursor-grabbing"
        >
          <Bars2Icon className="h-4 w-4" />
        </button>
      )}
      <div className="flex-1">{children}</div>
    </Tag>
  );
}
