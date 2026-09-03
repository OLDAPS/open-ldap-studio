/**
 * Turning an attribute value into something a row can show.
 *
 * Values cross the bridge as base64, because they are bytes on the Go side and
 * bytes are what the directory actually holds. Decoding is a presentation
 * concern and it is one-way: nothing here is ever sent back to a server. The
 * bytes stay available so an editor can round-trip exactly what it was given
 * (SC-007).
 */

/** The raw bytes of a base64 value. */
export function bytesOf(base64: string): Uint8Array {
  const binary = atob(base64);
  const out = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) out[i] = binary.charCodeAt(i);
  return out;
}

const utf8 = new TextDecoder('utf-8', { fatal: true });

export interface DisplayValue {
  /** What to render. For binary, a description rather than mojibake. */
  text: string;
  /** True when the value is not printable text and must not be shown as such. */
  binary: boolean;
  bytes: number;
}

/**
 * Decode a value for display.
 *
 * A value is treated as text only when it decodes as UTF-8 *and* carries no
 * control characters. Anything else is binary and is described rather than
 * printed: a JPEG rendered as text is a screenful of noise that hides the one
 * fact worth knowing, which is that it is a JPEG of a certain size.
 */
export function displayValue(base64: string): DisplayValue {
  let bytes: Uint8Array;
  try {
    bytes = bytesOf(base64);
  } catch {
    // Not valid base64 — show it as it arrived rather than inventing a value.
    return { text: base64, binary: false, bytes: base64.length };
  }

  const size = bytes.length;

  try {
    const text = utf8.decode(bytes);
    // Tab, newline and carriage return are the only control characters a
    // directory value legitimately carries; anything else means these bytes
    // are not text, whatever UTF-8 made of them.
    // eslint-disable-next-line no-control-regex -- that is precisely what is being detected
    if (/[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(text)) {
      return { text: describeBinary(bytes), binary: true, bytes: size };
    }
    return { text, binary: false, bytes: size };
  } catch {
    return { text: describeBinary(bytes), binary: true, bytes: size };
  }
}

/** "binary · JPEG · 24 KB" — the type where it is recognisable, and the size. */
function describeBinary(bytes: Uint8Array): string {
  const kind = sniff(bytes);
  return `binary${kind ? ` · ${kind}` : ''} · ${formatBytes(bytes.length)}`;
}

/** Enough magic numbers to name the formats a directory actually stores. */
function sniff(b: Uint8Array): string | undefined {
  if (b.length >= 3 && b[0] === 0xff && b[1] === 0xd8 && b[2] === 0xff) return 'JPEG';
  if (b.length >= 8 && b[0] === 0x89 && b[1] === 0x50 && b[2] === 0x4e && b[3] === 0x47) return 'PNG';
  if (b.length >= 4 && b[0] === 0x47 && b[1] === 0x49 && b[2] === 0x46) return 'GIF';
  // DER: a SEQUENCE, which is what a certificate and most ASN.1 starts with.
  if (b.length >= 2 && b[0] === 0x30 && (b[1] === 0x82 || b[1] === 0x81)) return 'DER';
  return undefined;
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 102.4) / 10} KB`;
  return `${Math.round(n / 104857.6) / 10} MB`;
}

/** A data: URL for a value known to be an image, for the photo panel. */
export function imageDataUrl(base64: string): string | undefined {
  const bytes = (() => {
    try {
      return bytesOf(base64);
    } catch {
      return undefined;
    }
  })();
  if (!bytes) return undefined;
  const kind = sniff(bytes);
  if (kind === 'JPEG') return `data:image/jpeg;base64,${base64}`;
  if (kind === 'PNG') return `data:image/png;base64,${base64}`;
  if (kind === 'GIF') return `data:image/gif;base64,${base64}`;
  return undefined;
}
