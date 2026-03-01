import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchView, fetchRepo, fetchStacks, fetchStack, fetchBranch } from "../api";

const mockFetch = vi.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockReset();
});

function mockOk(data: unknown) {
  mockFetch.mockResolvedValueOnce({
    ok: true,
    json: () => Promise.resolve(data),
  });
}

function mockError(status: number, statusText: string) {
  mockFetch.mockResolvedValueOnce({
    ok: false,
    status,
    statusText,
  });
}

describe("fetchView", () => {
  it("fetches from /api/view", async () => {
    const data = { repo: {}, stacks: [] };
    mockOk(data);

    const result = await fetchView();
    expect(result).toEqual(data);
    expect(mockFetch).toHaveBeenCalledWith("http://localhost:8080/api/view");
  });
});

describe("fetchRepo", () => {
  it("fetches from /api/repo", async () => {
    const data = { owner: "test", repo: "repo", trunk: "main", currentBranch: "main", remote: "origin" };
    mockOk(data);

    const result = await fetchRepo();
    expect(result).toEqual(data);
    expect(mockFetch).toHaveBeenCalledWith("http://localhost:8080/api/repo");
  });
});

describe("fetchStacks", () => {
  it("fetches from /api/stacks", async () => {
    mockOk([]);
    const result = await fetchStacks();
    expect(result).toEqual([]);
  });
});

describe("fetchStack", () => {
  it("encodes branch name in URL", async () => {
    mockOk({ rootBranch: "feat/foo", branches: [] });

    await fetchStack("feat/foo");
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/stacks/feat%2Ffoo"
    );
  });
});

describe("fetchBranch", () => {
  it("encodes branch name in URL", async () => {
    mockOk({ name: "feat/bar" });

    await fetchBranch("feat/bar");
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/branches/feat%2Fbar"
    );
  });
});

describe("error handling", () => {
  it("throws on non-ok response", async () => {
    mockError(404, "Not Found");

    await expect(fetchRepo()).rejects.toThrow("API error: 404 Not Found");
  });

  it("throws on 500 response", async () => {
    mockError(500, "Internal Server Error");

    await expect(fetchView()).rejects.toThrow("API error: 500 Internal Server Error");
  });
});
