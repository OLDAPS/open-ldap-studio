/**
 * The entry-info inspector (screen 1b, right).
 *
 * Facts about the entry that are not attributes to edit: the photo, the class
 * set, and when it last changed. Every fact comes from the entry the server
 * returned — an empty one says so rather than showing a plausible blank.
 */
import { displayValue, imageDataUrl } from './value';
import type { Attribute, Entry } from '@/bridge/types';

export function EntryInfoPanel({ entry, loading }: { entry?: Entry; loading: boolean }) {
  const photo = firstValue(entry?.attributes, 'jpegPhoto') ?? firstValue(entry?.attributes, 'photo');
  const photoUrl = photo ? imageDataUrl(photo) : undefined;

  const classes = valuesOf(entry?.attributes, 'objectClass');
  const modified = firstText(entry?.attributes, 'modifyTimestamp');
  const created = firstText(entry?.attributes, 'createTimestamp');
  const modifier = firstText(entry?.attributes, 'modifiersName');

  return (
    <aside className="inspector" aria-label="Entry info">
      <div className="inspector__header">
        <span>Entry info</span>
      </div>

      <div className="inspector__body">
        {photoUrl ? (
          <img
            src={photoUrl}
            alt="Entry photo"
            style={{
              width: '100%',
              borderRadius: 'var(--radius-chrome)',
              border: '1px solid var(--border-subtle)',
            }}
          />
        ) : (
          <div className="placeholder" style={{ height: 78 }}>
            {loading ? 'reading…' : entry ? 'no photo' : 'no entry selected'}
          </div>
        )}

        <Fact label="Object classes" value={classes.length ? classes.join(', ') : '—'} />
        {/* Timestamps are operational, so they appear only when those are on. */}
        <Fact label="Modified" value={modified ?? '—'} />
        <Fact label="Created" value={created ?? '—'} />
        {modifier ? <Fact label="By" value={modifier} /> : null}

        {!modified && entry ? (
          <p className="dim" style={{ margin: 0, fontSize: 'var(--text-micro)' }}>
            Turn on operational attributes to see timestamps.
          </p>
        ) : null}

        <button type="button" className="tag" style={{ alignSelf: 'flex-start' }} disabled={!entry}>
          Show in schema browser
        </button>
      </div>
    </aside>
  );
}

function find(attributes: Attribute[] | undefined, type: string): Attribute | undefined {
  return attributes?.find((a) => a.type.toLowerCase() === type.toLowerCase());
}

function firstValue(attributes: Attribute[] | undefined, type: string): string | undefined {
  return find(attributes, type)?.values?.[0];
}

function firstText(attributes: Attribute[] | undefined, type: string): string | undefined {
  const raw = firstValue(attributes, type);
  return raw ? displayValue(raw).text : undefined;
}

function valuesOf(attributes: Attribute[] | undefined, type: string): string[] {
  return (find(attributes, type)?.values ?? []).map((v) => displayValue(v).text);
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="dim" style={{ fontSize: 'var(--text-caption)' }}>
        {label}
      </div>
      <div className="mono" style={{ wordBreak: 'break-word' }}>
        {value}
      </div>
    </div>
  );
}
