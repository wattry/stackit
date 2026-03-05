/** Strip common prefixes like "user/timestamp/" to show a shorter name. */
export function shortenBranchName(name: string): string {
  const match = name.match(/^[^/]+\/\d{14}\/(.+)$/);
  return match ? match[1] : name;
}
