export function formatErrorTrace(
  error: Error & { digest?: string },
  extra?: Record<string, unknown>,
): string {
  const parts: string[] = [];
  parts.push(`${error.name}: ${error.message}`);
  if (error.digest) {
    parts.push(`digest: ${error.digest}`);
  }
  if (error.stack) {
    parts.push(error.stack);
  }
  if (extra && Object.keys(extra).length > 0) {
    parts.push(JSON.stringify(extra, null, 2));
  }
  return parts.join("\n\n");
}
