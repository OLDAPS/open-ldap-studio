import type { Attribute } from '@/bridge/types';

function line(description: string, bytes: string, base64: string): string {
  if (!bytes) return `${description}:`;
  if (!/^[\x20-\x7e]*$/.test(bytes) || /^[ :<]/.test(bytes) || bytes.endsWith(' ')) {
    return `${description}:: ${base64}`;
  }
  return `${description}: ${bytes}`;
}

/** RFC 2849 display records, preserving attribute order, options and raw bytes. */
export function entryLDIFLines(dn: string, attributes: Attribute[]): string[] {
  const dnBytes = Array.from(new TextEncoder().encode(dn), (byte) =>
    String.fromCharCode(byte),
  ).join('');
  const lines = [line('dn', dnBytes, btoa(dnBytes))];
  for (const attribute of attributes) {
    const description = [attribute.type, ...(attribute.options ?? [])].join(';');
    for (const encoded of attribute.values ?? []) {
      lines.push(line(description, atob(encoded), encoded));
    }
  }
  return lines;
}

export function serializeEntryLDIF(dn: string, attributes: Attribute[]): string {
  return `${entryLDIFLines(dn, attributes).join('\n')}\n\n`;
}
