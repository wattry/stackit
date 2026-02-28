// API client for stackit-web server

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// --- Response Types (matching Go API types) ---

export interface RepoResponse {
  owner: string;
  repo: string;
  trunk: string;
  currentBranch: string;
  remote: string;
  currentUser?: string;
}

export interface StackSummary {
  rootBranch: string;
  title: string;
  status: "shippable" | "pending" | "blocked" | "incomplete";
  scope?: string;
  branchCount: number;
  prCount: number;
  isCurrent: boolean;
  hasWorktree?: boolean;
  description?: string;
  owner?: string;
}

export interface StackDetail extends StackSummary {
  branches: BranchResponse[];
}

export interface BranchResponse {
  name: string;
  parent?: string;
  children?: string[];
  depth: number;
  isCurrent: boolean;
  needsRestack: boolean;
  isLocked: boolean;
  lockReason?: string;
  isFrozen: boolean;
  scope?: string;
  revision: string;
  commitDate: string;
  commitAuthor: string;
  commitCount: number;
  linesAdded: number;
  linesDeleted: number;
  commits?: CommitResponse[];
  pr?: PRResponse;
  ci?: CIResponse;
  remoteStatus?: RemoteStatus;
}

export interface PRResponse {
  number: number;
  title: string;
  state: "OPEN" | "MERGED" | "CLOSED";
  url: string;
  isDraft: boolean;
  base: string;
}

export interface CIResponse {
  status: "passing" | "failing" | "pending" | "none";
  reviewDecision: string;
  checks?: CheckDetailResponse[];
}

export interface CheckDetailResponse {
  name: string;
  status: string;
  conclusion: string;
}

export interface CommitResponse {
  sha: string;
  message: string;
}

export interface RemoteStatus {
  ahead: boolean;
  behind: boolean;
  diverged: boolean;
  missingRemote: boolean;
}

export interface TrunkCommitResponse {
  sha: string;
  message: string;
  author: string;
  date: string;
  kind: "regular" | "stack-merge";
  prNumber?: number;
  stackSize?: number;
  stackPRs?: number[];
  stackScope?: string;
}

// --- Combined View Response ---

export interface ViewResponse {
  repo: RepoResponse;
  stacks: StackDetail[];
  recentlyMerged?: TrunkCommitResponse[];
}

// --- Fetch Functions ---

async function fetchAPI<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

export function fetchView(): Promise<ViewResponse> {
  return fetchAPI<ViewResponse>("/api/view");
}

export function fetchRepo(): Promise<RepoResponse> {
  return fetchAPI<RepoResponse>("/api/repo");
}

export function fetchStacks(): Promise<StackSummary[]> {
  return fetchAPI<StackSummary[]>("/api/stacks");
}

export function fetchStack(rootBranch: string): Promise<StackDetail> {
  return fetchAPI<StackDetail>(`/api/stacks/${encodeURIComponent(rootBranch)}`);
}

export function fetchBranches(): Promise<BranchResponse[]> {
  return fetchAPI<BranchResponse[]>("/api/branches");
}

export function fetchBranch(name: string): Promise<BranchResponse> {
  return fetchAPI<BranchResponse>(
    `/api/branches/${encodeURIComponent(name)}`
  );
}
