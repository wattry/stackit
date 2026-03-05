export function stripPRSuffix(message: string): string {
  return message.replace(/\s*\(#\d+\)\s*$/, "");
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}
