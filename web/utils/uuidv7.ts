/**
 * UUID v7 (RFC 9562) — как oktopus/internal/id для id правил до первого sync.
 */
export function newUUIDv7(): string {
  const ms = Date.now();
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes.subarray(6));

  bytes[0] = Math.floor(ms / 0x10000000000) & 0xff;
  bytes[1] = Math.floor(ms / 0x100000000) & 0xff;
  bytes[2] = Math.floor(ms / 0x1000000) & 0xff;
  bytes[3] = Math.floor(ms / 0x10000) & 0xff;
  bytes[4] = Math.floor(ms / 0x100) & 0xff;
  bytes[5] = ms & 0xff;

  bytes[6] = (bytes[6] & 0x0f) | 0x70;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;

  const hex = [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

export function ensureRuleId(id: string | undefined): string {
  const trimmed = id?.trim();
  return trimmed ? trimmed : newUUIDv7();
}
