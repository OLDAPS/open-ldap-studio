/**
 * The two pure helpers the tree and entry editor are built on.
 *
 * Both have a failure mode that is invisible until it matters: an RDN split on
 * the wrong comma silently mislabels every row under a DN containing an
 * escaped one, and a binary value decoded as text fills a row with noise.
 */
import { describe, expect, it } from 'vitest';

import { parentOf, rdnOf } from '@/views/browser/dn';
import { displayValue } from '@/views/browser/value';

describe('dn', () => {
  it('splits on the first separator', () => {
    expect(rdnOf('cn=jrivera,ou=people,dc=example,dc=org')).toBe('cn=jrivera');
    expect(parentOf('cn=jrivera,ou=people,dc=example,dc=org')).toBe('ou=people,dc=example,dc=org');
  });

  it('keeps an escaped comma inside the RDN', () => {
    // Without this, "Doe\, Jane" becomes "cn=Doe\" and the tree lies.
    expect(rdnOf('cn=Doe\\, Jane,ou=people,dc=example,dc=org')).toBe('cn=Doe\\, Jane');
    expect(parentOf('cn=Doe\\, Jane,ou=people,dc=example,dc=org')).toBe(
      'ou=people,dc=example,dc=org',
    );
  });

  it('treats a single-component DN as its own RDN', () => {
    expect(rdnOf('dc=org')).toBe('dc=org');
    expect(parentOf('dc=org')).toBe('');
  });

  it('does not rewrite what it returns', () => {
    // Case and spacing are the server's, and survive untouched.
    const dn = 'CN=Jordan Rivera, OU=People,DC=Example,DC=org';
    expect(rdnOf(dn)).toBe('CN=Jordan Rivera');
    expect(`${rdnOf(dn)},${parentOf(dn)}`).toBe(dn);
  });
});

const b64 = (text: string) => btoa(String.fromCharCode(...new TextEncoder().encode(text)));

describe('displayValue', () => {
  it('decodes UTF-8 text', () => {
    const value = displayValue(b64('Tomás Vásquez'));
    expect(value.text).toBe('Tomás Vásquez');
    expect(value.binary).toBe(false);
  });

  it('keeps a leading space', () => {
    // The fixture stores one deliberately; losing it here would hide a real
    // round-trip bug elsewhere.
    expect(displayValue(b64(' leading space')).text).toBe(' leading space');
  });

  it('describes a JPEG rather than printing it', () => {
    const jpeg = btoa(String.fromCharCode(0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46));
    const value = displayValue(jpeg);
    expect(value.binary).toBe(true);
    expect(value.text).toContain('JPEG');
    expect(value.text).toContain('8 B');
  });

  it('treats a value with control characters as binary', () => {
    const raw = btoa(String.fromCharCode(0x01, 0x02, 0x03, 0x04));
    expect(displayValue(raw).binary).toBe(true);
  });

  it('allows tabs and newlines in text', () => {
    const value = displayValue(b64('line one\nline two\tindented'));
    expect(value.binary).toBe(false);
    expect(value.text).toContain('\n');
  });

  it('reports an empty value as empty text, not as binary', () => {
    const value = displayValue('');
    expect(value.binary).toBe(false);
    expect(value.text).toBe('');
  });
});
