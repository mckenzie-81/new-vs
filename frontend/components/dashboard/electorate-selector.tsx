"use client";

import { useEffect, useState } from "react";
import { listElectorateDatabases, getDistinctValues, ElectorateDatabase } from "@/lib/api";
import { Select } from "@/components/ui/select";
import { Field } from "@/components/ui/field";

const MAX_VERIFICATION_COLUMNS = 2;

export function ElectorateSelector({
  databaseId,
  filter,
  verificationColumns,
  onDatabaseChange,
  onFilterChange,
  onVerificationColumnsChange,
}: {
  databaseId: string;
  filter: Record<string, string>;
  verificationColumns: string[];
  onDatabaseChange: (id: string) => void;
  onFilterChange: (filter: Record<string, string>) => void;
  onVerificationColumnsChange: (columns: string[]) => void;
}) {
  const [databases, setDatabases] = useState<ElectorateDatabase[] | null>(null);
  const [distinctValues, setDistinctValues] = useState<Record<string, string[]>>({});

  useEffect(() => {
    listElectorateDatabases()
      .then(setDatabases)
      .catch(() => setDatabases([]));
  }, []);

  const selectedDb = databases?.find((d) => d.id === databaseId);
  const sensitiveColumns = selectedDb?.sensitiveColumns ?? [];
  const isSensitive = (c: string) => sensitiveColumns.some((s) => s.toLowerCase() === c.toLowerCase());
  const filterableColumns = (selectedDb?.columns ?? []).filter(
    (c) => c.toLowerCase() !== "name" && c.toLowerCase() !== "email" && !isSensitive(c)
  );
  // Unlike filterableColumns, verification can use a hashed column (bcrypt
  // compare) or a plain one (direct compare) - plain just proves less,
  // since its value is already visible to any systemadmin. That's a
  // tradeoff for the admin to weigh, not something to hide by excluding it.
  const verifiableColumns = (selectedDb?.columns ?? []).filter(
    (c) => c.toLowerCase() !== "name" && c.toLowerCase() !== "email"
  );
  const rows = Object.entries(filter);

  function toggleVerificationColumn(column: string) {
    if (verificationColumns.includes(column)) {
      onVerificationColumnsChange(verificationColumns.filter((c) => c !== column));
    } else if (verificationColumns.length < MAX_VERIFICATION_COLUMNS) {
      onVerificationColumnsChange([...verificationColumns, column]);
    }
  }

  function loadDistinctValues(column: string) {
    if (!databaseId || distinctValues[column]) return;
    getDistinctValues(databaseId, column)
      .then((values) => setDistinctValues((prev) => ({ ...prev, [column]: values })))
      .catch(() => setDistinctValues((prev) => ({ ...prev, [column]: [] })));
  }

  function addFilterRow() {
    const unused = filterableColumns.find((c) => !(c in filter));
    if (!unused) return;
    loadDistinctValues(unused);
    onFilterChange({ ...filter, [unused]: "" });
  }

  function updateRowColumn(oldColumn: string, newColumn: string) {
    const next = { ...filter };
    delete next[oldColumn];
    next[newColumn] = "";
    loadDistinctValues(newColumn);
    onFilterChange(next);
  }

  function updateRowValue(column: string, value: string) {
    onFilterChange({ ...filter, [column]: value });
  }

  function removeRow(column: string) {
    const next = { ...filter };
    delete next[column];
    onFilterChange(next);
  }

  return (
    <div className="flex flex-col gap-4">
      <Field
        label="Electorate database"
        htmlFor="electorate-database"
        hint="Who's eligible to vote in this election. Optional for now - add one before submitting for approval."
      >
        <Select
          id="electorate-database"
          value={databaseId}
          onChange={(e) => {
            onDatabaseChange(e.target.value);
            onFilterChange({});
            onVerificationColumnsChange([]);
          }}
        >
          <option value="">
            {databases === null ? "Loading..." : "No electorate database yet"}
          </option>
          {databases?.map((db) => (
            <option key={db.id} value={db.id}>
              {db.name} ({db.voterCount} voters)
            </option>
          ))}
        </Select>
      </Field>

      {selectedDb && filterableColumns.length > 0 && (
        <div>
          <p className="text-sm font-semibold text-ink">Narrow eligibility</p>
          <p className="mt-1 text-sm text-ink-muted">
            Optional. Leave blank to make everyone in this database eligible, or add a filter to
            restrict it - e.g. just one department or year group.
          </p>

          {rows.length > 0 && (
            <div className="mt-3 flex flex-col gap-2">
              {rows.map(([column, value]) => (
                <div key={column} className="flex items-end gap-2">
                  <Select
                    value={column}
                    onChange={(e) => updateRowColumn(column, e.target.value)}
                    className="flex-1"
                  >
                    {filterableColumns.map((c) => (
                      <option key={c} value={c} disabled={c !== column && c in filter}>
                        {c}
                      </option>
                    ))}
                  </Select>
                  <Select
                    value={value}
                    onChange={(e) => updateRowValue(column, e.target.value)}
                    className="flex-1"
                  >
                    <option value="" disabled>
                      Choose a value
                    </option>
                    {(distinctValues[column] ?? []).map((v) => (
                      <option key={v} value={v}>
                        {v}
                      </option>
                    ))}
                  </Select>
                  <button
                    type="button"
                    onClick={() => removeRow(column)}
                    className="mb-2 shrink-0 text-sm text-ink-faint hover:text-danger"
                  >
                    Remove
                  </button>
                </div>
              ))}
            </div>
          )}

          {rows.length < filterableColumns.length && (
            <button
              type="button"
              onClick={addFilterRow}
              className="mt-3 text-sm font-semibold text-accent hover:underline"
            >
              + Add filter
            </button>
          )}
        </div>
      )}

      {selectedDb && verifiableColumns.length > 0 && (
        <div>
          <p className="text-sm font-semibold text-ink">Voter verification</p>
          <p className="mt-1 text-sm text-ink-muted">
            Optional, up to {MAX_VERIFICATION_COLUMNS}. A voter must also enter the correct value
            for each column chosen here before they can vote - on top of clicking their emailed
            link. A hashed column is a real secret nobody can look up; a plain one is just
            whatever&apos;s in that column, visible to any systemadmin already - weaker, but
            better than nothing if this database has no hashed column.
          </p>
          <div className="mt-3 flex flex-col gap-2">
            {verifiableColumns.map((column) => {
              const checked = verificationColumns.includes(column);
              const disabled = !checked && verificationColumns.length >= MAX_VERIFICATION_COLUMNS;
              return (
                <label key={column} className="flex items-center gap-2 text-sm text-ink">
                  <input
                    type="checkbox"
                    checked={checked}
                    disabled={disabled}
                    onChange={() => toggleVerificationColumn(column)}
                  />
                  {column}
                  <span className="text-xs text-ink-faint">
                    {isSensitive(column) ? "(hashed)" : "(plain text)"}
                  </span>
                </label>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
