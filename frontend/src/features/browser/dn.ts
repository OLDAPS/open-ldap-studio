/**
 * Display-only helpers for distinguished names.
 *
 * Nothing here rewrites a DN. The server's rendering is the one the
 * application sends back, so these functions only ever return a *slice* of the
 * string they were given, never a re-encoded form (FR-019, FR-020, Principle
 * III). Splitting is done on an unescaped comma so that a DN carrying an
 * escaped one — `cn=Doe\, Jane,ou=people` — keeps it.
 */

/** The index of the first unescaped comma, or -1. */
function firstSeparator(dn: string): number {
  for (let i = 0; i < dn.length; i += 1) {
    if (dn[i] === '\\') {
      i += 1; // skip the escaped character
      continue;
    }
    if (dn[i] === ',') return i;
  }
  return -1;
}

/**
 * The relative distinguished name — what a tree row shows.
 *
 * A DN with no separator is its own RDN, which is what a naming context looks
 * like when it has a single component.
 */
export function rdnOf(dn: string): string {
  const at = firstSeparator(dn);
  return at === -1 ? dn : dn.slice(0, at);
}

/** Everything after the RDN, or '' at the top of a tree. */
export function parentOf(dn: string): string {
  const at = firstSeparator(dn);
  return at === -1 ? '' : dn.slice(at + 1);
}
