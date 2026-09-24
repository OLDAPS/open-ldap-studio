import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
/**
 * The search editor (screen 1c, document area).
 *
 * Base, filter, returning attributes, scope and limits are one card, because
 * they are one request: a filter read without its scope and size limit is a
 * filter whose cost you cannot judge.
 *
 * The raw RFC 4515 field is the primary input and the builder is a way of
 * writing into it, not a parallel model — so what is run is always what is on
 * screen.
 */
import { useState } from 'react';

import { ResultGrid } from './ResultGrid';

export function SearchEditor() {
  const [filter, setFilter] = useState('(objectClass=*)');

  return (
    <>
      <div className="pane" style={{ flex: 'none' }}>
        <Card className="card card--tight">
          <Field orientation="horizontal" className="field-row">
            <FieldLabel className="field-row__label" htmlFor="search-base">
              Search base
            </FieldLabel>
            <Input id="search-base" className="field field--mono" placeholder="dc=example,dc=com" />
            <Button type="button" variant="outline">
              Browse…
            </Button>
          </Field>

          <Field orientation="horizontal" className="field-row">
            <FieldLabel className="field-row__label" htmlFor="search-filter">
              Filter
            </FieldLabel>
            <Input
              id="search-filter"
              className="field field--mono"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              spellCheck={false}
            />
            <Button type="button" variant="outline">
              Builder
            </Button>
            <Button type="button" variant="outline">
              Validate
            </Button>
          </Field>

          <Field orientation="horizontal" className="field-row">
            <FieldLabel className="field-row__label" htmlFor="search-attrs">
              Returning attrs
            </FieldLabel>
            <Input id="search-attrs" className="field field--mono" placeholder="uid, cn, mail, +" />
          </Field>

          <Field orientation="horizontal" className="field-row field-row--wrap">
            <span className="field-row__label">Scope / limits</span>
            <NativeSelect
              className="field"
              defaultValue="sub"
              aria-label="Scope"
              style={{ width: 120 }}
            >
              <option value="base">base</option>
              <option value="one">one level</option>
              <option value="sub">subtree</option>
            </NativeSelect>
            <Input
              className="field"
              aria-label="Size limit"
              defaultValue="count 1000"
              style={{ width: 110 }}
            />
            <Input
              className="field"
              aria-label="Time limit"
              defaultValue="time 10 s"
              style={{ width: 100 }}
            />
            <NativeSelect
              className="field"
              defaultValue="find"
              aria-label="Aliases"
              style={{ width: 150 }}
            >
              <option value="never">aliases: never</option>
              <option value="find">aliases: finding</option>
              <option value="search">aliases: searching</option>
              <option value="always">aliases: always</option>
            </NativeSelect>
            <NativeSelect
              className="field"
              defaultValue="follow"
              aria-label="Referrals"
              style={{ width: 150 }}
            >
              <option value="follow">referrals: follow</option>
              <option value="ignore">referrals: ignore</option>
              <option value="ask">referrals: ask</option>
            </NativeSelect>

            <span className="actions actions--end">
              <Button type="button" variant="outline">
                Save search
              </Button>
              <Button type="button" variant="default">
                Run
              </Button>
            </span>
          </Field>
        </Card>
      </div>

      <ResultGrid />
    </>
  );
}
