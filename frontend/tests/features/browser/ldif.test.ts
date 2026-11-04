import { expect, it } from 'vitest';
import { entryLDIFLines, serializeEntryLDIF } from '@/features/browser/ldif';

it('serializes options, empty values and unsafe LDIF bytes without loss', () => {
  const values = [' leading', ':colon', '<url', 'trailing ', 'a\nb', 'Á', '', 'ordinary'];
  const encoded = values.map((value) => btoa(String.fromCharCode(...new TextEncoder().encode(value))));
  const attributes = [{ type: 'description', options: ['lang-en'], values: encoded, isOperational: false }];
  const lines = entryLDIFLines('cn=subject,dc=example', attributes);
  expect(lines[0]).toBe('dn: cn=subject,dc=example');
  for (let index = 0; index < 6; index++) {
    expect(lines[index + 1]).toBe(`description;lang-en:: ${encoded[index]}`);
  }
  expect(lines[7]).toBe('description;lang-en:');
  expect(lines[8]).toBe('description;lang-en: ordinary');
  expect(serializeEntryLDIF('cn=subject,dc=example', attributes)).toBe(`${lines.join('\n')}\n\n`);
});
