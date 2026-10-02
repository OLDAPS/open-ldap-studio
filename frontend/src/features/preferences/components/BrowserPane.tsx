import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
/**
 * Screen 5a — Browser & tree.
 *
 * Paging, limits and alias/referral handling are on the same pane as the
 * display options because they are the same decision: how much of a directory
 * this application will pull before it asks.
 */
export function BrowserPane() {
  return (
    <>
      <Card className="card">
        <span className="card__label">Paging &amp; limits</span>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-page">
            Entries per page
          </FieldLabel>
          <Input
            id="pref-page"
            className="field"
            type="number"
            defaultValue={100}
            style={{ width: 90, flex: 'none' }}
          />
          <FieldLabel className="field-row__label field-row__label--auto" htmlFor="pref-fetch">
            Fetch on scroll
          </FieldLabel>
          <Checkbox id="pref-fetch" />
          <span className="dim">off means an explicit &ldquo;fetch next&rdquo; row</span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-size">
            Size limit
          </FieldLabel>
          <Input
            id="pref-size"
            className="field"
            type="number"
            defaultValue={1000}
            style={{ width: 90, flex: 'none' }}
          />
          <FieldLabel className="field-row__label field-row__label--auto" htmlFor="pref-time">
            Time limit
          </FieldLabel>
          <Input
            id="pref-time"
            className="field"
            defaultValue="10 s"
            style={{ width: 90, flex: 'none' }}
          />
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-aliases">
            Aliases · referrals
          </FieldLabel>
          <NativeSelect id="pref-aliases" className="field" defaultValue="find">
            <option value="never">dereference: never</option>
            <option value="find">dereference: finding</option>
            <option value="search">dereference: searching</option>
            <option value="always">dereference: always</option>
          </NativeSelect>
          <NativeSelect className="field" defaultValue="follow" aria-label="Referrals">
            <option value="follow">referrals: follow</option>
            <option value="ignore">referrals: ignore</option>
            <option value="ask">referrals: ask</option>
          </NativeSelect>
        </Field>
      </Card>

      <Card className="card">
        <span className="card__label">Display</span>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Entry label</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              RDN
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              full DN
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              attribute…
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-sort">
            Sort children by
          </FieldLabel>
          <NativeSelect id="pref-sort" className="field" defaultValue="rdn">
            <option value="rdn">RDN</option>
            <option value="none">server order</option>
          </NativeSelect>
        </Field>
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Show</span>
          <span className="tag-set">
            <Button variant="outline" size="xs" type="button" className="tag" data-selected>
              child count
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              operational attrs
            </Button>
            <Button variant="outline" size="xs" type="button" className="tag">
              subentries
            </Button>
          </span>
        </Field>
        <Field orientation="horizontal" className="field-row">
          <FieldLabel className="field-row__label" htmlFor="pref-expand">
            Expand on connect
          </FieldLabel>
          <NativeSelect id="pref-expand" className="field" defaultValue="first">
            <option value="first">first naming context</option>
            <option value="none">none</option>
            <option value="last">last session</option>
          </NativeSelect>
        </Field>
      </Card>
    </>
  );
}
