export const ERROR_CODE_PATTERN = /^[a-z][a-z0-9_]*$/;

export function isApiErrorCode(value: string): boolean {
  return ERROR_CODE_PATTERN.test(value);
}
