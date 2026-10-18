"use client";

import { useRef, useState } from "react";
import { ApiError, UploadKind, uploadFile } from "@/lib/api";
import { SecondaryButton } from "@/components/ui/button";

const MAX_BYTES = 5 * 1024 * 1024;
const ACCEPT = "image/png,image/jpeg,image/webp,image/gif,image/svg+xml";

export function ImageUpload({
  id,
  kind,
  value,
  onChange,
}: {
  id: string;
  kind: UploadKind;
  value: string;
  onChange: (url: string) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleFile(file: File) {
    setError(null);
    if (file.size > MAX_BYTES) {
      setError("That file is too large - 5MB max.");
      return;
    }
    setUploading(true);
    try {
      const url = await uploadFile(kind, file);
      onChange(url);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Upload failed. Please try again.");
    } finally {
      setUploading(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  return (
    <div>
      <div className="flex items-center gap-3">
        {value ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={value}
            alt=""
            className="h-16 w-16 shrink-0 border border-line bg-page object-contain"
          />
        ) : (
          <div className="flex h-16 w-16 shrink-0 items-center justify-center border border-dashed border-line text-xs text-ink-faint">
            None
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <div className="flex gap-2">
            <SecondaryButton
              type="button"
              onClick={() => inputRef.current?.click()}
              disabled={uploading}
            >
              {uploading ? "Uploading..." : value ? "Change" : "Upload image"}
            </SecondaryButton>
            {value && (
              <SecondaryButton type="button" tone="danger" onClick={() => onChange("")}>
                Remove
              </SecondaryButton>
            )}
          </div>
          <p className="text-xs text-ink-faint">PNG, JPEG, WEBP, GIF, or SVG - 5MB max.</p>
        </div>
      </div>

      <input
        ref={inputRef}
        id={id}
        type="file"
        accept={ACCEPT}
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) handleFile(file);
        }}
      />

      {error && <p className="mt-1.5 text-sm text-danger">{error}</p>}
    </div>
  );
}
