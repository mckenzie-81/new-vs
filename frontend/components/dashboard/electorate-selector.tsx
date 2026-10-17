"use client";

import { useEffect, useState } from "react";
import { listElectorateDatabases, getDistinctValues, ElectorateDatabase } from "@/lib/api";
import { Select } from "@/components/ui/select";
import { Field } from "@/components/ui/field";

export function ElectorateSelector({
  databaseId,
  filter,
  onDatabaseChange,
  onFilterChange,
}: {
  databaseId: string;
  filter: Record<string, string>;
  onDatabaseChange: (id: string) => void;
  onFilterChange: (filter: Record<string, string>) => void;
}) {
  const [databases, setDatabases] = useState<ElectorateDatabase[] | null>(null);
  const [distinctValues, setDistinctValues] = useState<Record<string, string[]>>({});

  useEffect(() => {
    listElectorateDatabases()
      .then(setDatabases)
      .catch(() => setDatabases([]));
  }, []);

  const selectedDb = databases?.find((d) => d.id === databaseId);
  const filterableColumns = (selectedDb?.columns ?? []).filter(
    (c) => c.toLowerCase() !== "name" && c.toLowerCase() !== "email"
  );
  const rows = Object.entries(filter);

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
        hint="Who's eligible to vote in this election."
      >
        <Select
          id="electorate-database"
          required
          value={databaseId}
          onChange={(e) => {
            onDatabaseChange(e.target.value);
            onFilterChange({});
          }}
        >
          <option value="" disabled>
            {databases === null ? "Loading..." : "Choose a database"}
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
    </div>
  );
}
