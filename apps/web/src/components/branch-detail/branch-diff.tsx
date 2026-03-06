"use client";

import {
  parsePatchFiles,
  type FileDiffMetadata,
} from "@pierre/diffs";
import { FileDiff } from "@pierre/diffs/react";
import { ChevronDown, ChevronRight, FileText, Folder, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { fetchBranchDiff, type BranchDiffResponse } from "@/lib/api";
import { cn } from "@/lib/utils";

interface BranchDiffProps {
  branchName: string;
  revision: string;
  onExit?: () => void;
}

const DIFF_OPTIONS = {
  diffStyle: "split",
  lineDiffType: "word",
  hunkSeparators: "metadata",
  overflow: "scroll",
  themeType: "system",
} as const;

function getFileKey(file: FileDiffMetadata, index: number): string {
  return `${file.prevName ?? file.name}:${file.name}:${index}`;
}

function getFileLabel(file: FileDiffMetadata): string {
  if (file.prevName && file.prevName !== file.name) {
    return `${file.prevName} -> ${file.name}`;
  }
  return file.name;
}

function hasLineChanges(file: FileDiffMetadata): boolean {
  return file.hunks.some((hunk) =>
    hunk.hunkContent.some(
      (content) =>
        content.type === "change" &&
        (content.additions.length > 0 || content.deletions.length > 0)
    )
  );
}

function getChangeTypeIcon(file: FileDiffMetadata) {
  const colorClass =
    file.type === "new"
      ? "text-green-600"
      : file.type === "deleted"
        ? "text-red-600"
        : "text-blue-600";

  const iconType =
    file.type === "rename-pure" || file.type === "rename-changed"
      ? "rename-changed"
      : file.type;

  return (
    <svg
      className={`h-3.5 w-3.5 shrink-0 ${colorClass}`}
      data-change-icon={file.type}
      aria-hidden="true"
      viewBox="0 0 16 16"
    >
      {iconType === "new" && (
        <>
          <path d="M8 4a.75.75 0 0 1 .75.75v2.5h2.5a.75.75 0 0 1 0 1.5h-2.5v2.5a.75.75 0 0 1-1.5 0v-2.5h-2.5a.75.75 0 0 1 0-1.5h2.5v-2.5A.75.75 0 0 1 8 4" />
          <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8" />
        </>
      )}
      {iconType === "deleted" && (
        <>
          <path d="M4 8a.75.75 0 0 1 .75-.75h6.5a.75.75 0 0 1 0 1.5h-6.5A.75.75 0 0 1 4 8" />
          <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8" />
        </>
      )}
      {iconType === "rename-changed" && (
        <>
          <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8" />
          <path d="M8.495 4.695a.75.75 0 0 0-.05 1.06L10.486 8l-2.041 2.246a.75.75 0 0 0 1.11 1.008l2.5-2.75a.75.75 0 0 0 0-1.008l-2.5-2.75a.75.75 0 0 0-1.06-.051m-4 0a.75.75 0 0 0-.05 1.06l2.044 2.248-1.796 1.995a.75.75 0 0 0 1.114 1.004l2.25-2.5a.75.75 0 0 0-.002-1.007l-2.5-2.75a.75.75 0 0 0-1.06-.05" />
        </>
      )}
      {iconType === "change" && (
        <>
          <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706s.826.607 1.706.802c.898.2 2.091.288 3.704.288s2.806-.088 3.704-.288c.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5s-2.806.088-3.704.288c-.88.196-1.381.478-1.706.802s-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8M0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m8 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6" />
        </>
      )}
    </svg>
  );
}

interface ExplorerFileEntry {
  key: string;
  title: string;
  path: string;
  name: string;
}

interface ExplorerFolderNode {
  name: string;
  path: string;
  folders: Map<string, ExplorerFolderNode>;
  files: ExplorerFileEntry[];
}

interface ExplorerFolderRow {
  kind: "folder";
  key: string;
  name: string;
  depth: number;
}

interface ExplorerFileRow {
  kind: "file";
  key: string;
  fileKey: string;
  depth: number;
  name: string;
  title: string;
}

type ExplorerRow = ExplorerFolderRow | ExplorerFileRow;

function createFolderNode(name: string, path: string): ExplorerFolderNode {
  return {
    name,
    path,
    folders: new Map<string, ExplorerFolderNode>(),
    files: [],
  };
}

function buildExplorerRows(entries: ExplorerFileEntry[]): ExplorerRow[] {
  const root = createFolderNode("", "");
  const sortedEntries = [...entries].sort((a, b) => a.path.localeCompare(b.path));

  sortedEntries.forEach((entry) => {
    const segments = entry.path.split("/").filter(Boolean);
    const fileName = segments.at(-1) ?? entry.path;
    let node = root;
    let currentPath = "";

    segments.slice(0, -1).forEach((segment) => {
      currentPath = currentPath ? `${currentPath}/${segment}` : segment;
      let folder = node.folders.get(segment);
      if (!folder) {
        folder = createFolderNode(segment, currentPath);
        node.folders.set(segment, folder);
      }
      node = folder;
    });

    node.files.push({
      ...entry,
      name: fileName,
    });
  });

  const rows: ExplorerRow[] = [];

  function walk(node: ExplorerFolderNode, depth: number) {
    const folders = Array.from(node.folders.values()).sort((a, b) =>
      a.name.localeCompare(b.name)
    );
    const files = [...node.files].sort((a, b) => a.name.localeCompare(b.name));

    folders.forEach((folder) => {
      rows.push({
        kind: "folder",
        key: `folder:${folder.path}`,
        name: folder.name,
        depth,
      });
      walk(folder, depth + 1);
    });

    files.forEach((file) => {
      rows.push({
        kind: "file",
        key: `file:${file.key}`,
        fileKey: file.key,
        depth,
        name: file.name,
        title: file.title,
      });
    });
  }

  walk(root, 0);
  return rows;
}

export function BranchDiff({ branchName, revision, onExit }: BranchDiffProps) {
  const [diff, setDiff] = useState<BranchDiffResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [collapsedFiles, setCollapsedFiles] = useState<Record<string, boolean>>(
    {}
  );
  const [activeFileKey, setActiveFileKey] = useState<string | null>(null);
  const fileSectionRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const diffScrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let active = true;
    queueMicrotask(() => {
      if (!active) return;
      setLoading(true);
      setError(null);
      setDiff(null);
    });

    fetchBranchDiff(branchName)
      .then((result) => {
        if (!active) return;
        setDiff(result);
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Failed to load branch diff");
      })
      .finally(() => {
        if (!active) return;
        setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [branchName, revision]);

  const parsed = useMemo(() => {
    if (!diff?.patch.trim()) {
      return { files: [] as FileDiffMetadata[], parseError: null as string | null };
    }

    try {
      const files = parsePatchFiles(diff.patch).flatMap((group) => group.files);
      return { files, parseError: null as string | null };
    } catch (err) {
      return {
        files: [] as FileDiffMetadata[],
        parseError: err instanceof Error ? err.message : "Failed to parse patch",
      };
    }
  }, [diff]);

  const visibleCollapsedFiles = useMemo(() => {
    const next: Record<string, boolean> = {};
    parsed.files.forEach((file, index) => {
      const key = getFileKey(file, index);
      if (collapsedFiles[key]) next[key] = true;
    });
    return next;
  }, [collapsedFiles, parsed.files]);

  const explorerRows = useMemo(
    () =>
      buildExplorerRows(
        parsed.files.map((file, index) => ({
          key: getFileKey(file, index),
          title: getFileLabel(file),
          path: file.name,
          name: file.name,
        }))
      ),
    [parsed.files]
  );

  const fileKeys = useMemo(
    () => parsed.files.map((file, index) => getFileKey(file, index)),
    [parsed.files]
  );
  const selectedFileKey =
    activeFileKey && fileKeys.includes(activeFileKey)
      ? activeFileKey
      : (fileKeys[0] ?? null);

  useEffect(() => {
    if (parsed.files.length === 0) return;
    const diffScrollRoot = diffScrollRef.current;
    const observerRoot =
      diffScrollRoot && diffScrollRoot.scrollHeight > diffScrollRoot.clientHeight
        ? diffScrollRoot
        : null;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
        const key = visible?.target.getAttribute("data-file-key");
        if (key) setActiveFileKey(key);
      },
      {
        root: observerRoot,
        rootMargin: "-15% 0px -70% 0px",
        threshold: [0, 0.2, 0.6, 1],
      }
    );

    parsed.files.forEach((file, index) => {
      const key = getFileKey(file, index);
      const node = fileSectionRefs.current[key];
      if (node) observer.observe(node);
    });

    return () => {
      observer.disconnect();
    };
  }, [parsed.files]);

  const fileCountLabel = `${parsed.files.length} file${parsed.files.length !== 1 ? "s" : ""}`;
  const showFileCount = !loading && !error && !parsed.parseError;
  const workspaceMode = Boolean(onExit);
  const contentLayoutClass = workspaceMode
    ? "flex min-h-0 flex-1 items-stretch gap-4"
    : "grid gap-3 lg:grid-cols-[18rem,minmax(0,1fr)] lg:items-start lg:gap-4";
  const explorerClass = workspaceMode
    ? "flex min-h-0 w-72 shrink-0 flex-col overflow-hidden rounded-md border bg-card"
    : "rounded-md border bg-card/60 lg:flex lg:max-h-[calc(100vh-14rem)] lg:min-h-0 lg:flex-col";
  const explorerListClass = workspaceMode
    ? "h-full overflow-y-auto p-2"
    : "max-h-56 overflow-auto p-2 lg:max-h-none lg:flex-1";
  const diffPaneClass = workspaceMode
    ? "min-h-0 min-w-0 overflow-y-auto pr-1"
    : "min-w-0 lg:max-h-[calc(100vh-14rem)] lg:overflow-y-auto lg:pr-1";
  const rootClassName = workspaceMode ? "flex h-full min-h-0 flex-col" : "space-y-2";

  return (
    <div className={rootClassName}>
      <div
        className={
          workspaceMode
            ? "mb-2 flex shrink-0 items-center justify-between gap-2 border-b px-1 pb-3"
            : "flex items-center justify-between gap-2"
        }
      >
        {workspaceMode ? (
          <>
            {showFileCount ? (
              <span className="text-xs text-muted-foreground">{fileCountLabel}</span>
            ) : (
              <span />
            )}
            <button
              onClick={onExit}
              className="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors"
            >
              <X className="h-3.5 w-3.5" />
              Back to all stacks
            </button>
          </>
        ) : (
          <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            Diff
          </h4>
        )}
        {!workspaceMode && showFileCount && (
          <span className="text-xs text-muted-foreground">{fileCountLabel}</span>
        )}
      </div>

      {loading && (
        <p className="text-sm text-muted-foreground">Loading diff…</p>
      )}

      {!loading && error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {!loading && !error && diff && !diff.patch.trim() && (
        <p className="text-sm text-muted-foreground">
          No changes relative to base commit {diff.baseRevision.slice(0, 7)}.
        </p>
      )}

      {!loading && !error && parsed.parseError && diff?.patch && (
        <div className="space-y-2">
          <p className="text-sm text-destructive">
            Could not parse diff. Showing raw patch.
          </p>
          <pre className="max-h-80 overflow-auto rounded-md border bg-muted/30 p-3 text-xs">
            {diff.patch}
          </pre>
        </div>
      )}

      {!loading && !error && !parsed.parseError && parsed.files.length > 0 && (
        <div className={contentLayoutClass}>
          <aside className={explorerClass}>
            <div className="border-b px-3 py-2 text-xs font-medium text-muted-foreground">
              Files
            </div>
            <div className={explorerListClass}>
              <ul className="space-y-0.5">
                {explorerRows.map((row) => (
                  <li key={row.key}>
                    {row.kind === "folder" ? (
                      <div
                        className="flex items-center gap-1.5 rounded px-2 py-1 text-xs text-muted-foreground"
                        style={{ paddingLeft: `${row.depth * 14 + 8}px` }}
                      >
                        <Folder className="h-3.5 w-3.5 shrink-0" />
                        <span className="truncate">{row.name}</span>
                      </div>
                    ) : (
                      <button
                        onClick={() => {
                          setActiveFileKey(row.fileKey);
                          fileSectionRefs.current[row.fileKey]?.scrollIntoView({
                            behavior: "smooth",
                            block: "start",
                          });
                        }}
                        className={cn(
                          "flex w-full items-center gap-1.5 rounded py-1 pr-2 text-left font-mono text-xs transition-colors",
                          selectedFileKey === row.fileKey
                            ? "bg-primary/10 text-foreground"
                            : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                        )}
                        style={{ paddingLeft: `${row.depth * 14 + 8}px` }}
                        title={row.title}
                      >
                        <FileText className="h-3.5 w-3.5 shrink-0" />
                        <span className="truncate">{row.name}</span>
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          </aside>

          <div ref={diffScrollRef} className={diffPaneClass}>
            <div className="space-y-3">
              {parsed.files.map((file, i) => {
                const fileKey = getFileKey(file, i);
                const collapsible = hasLineChanges(file);

                return (
                  <div
                    key={fileKey}
                    ref={(node) => {
                      fileSectionRefs.current[fileKey] = node;
                    }}
                    data-file-key={fileKey}
                    className={cn(
                      "relative rounded-md border bg-background scroll-mt-24",
                      selectedFileKey === fileKey && "ring-1 ring-primary/20"
                    )}
                  >
                    {collapsible && visibleCollapsedFiles[fileKey] ? (
                      <div className="flex items-center justify-between gap-3 px-3 py-2">
                        <div className="flex min-w-0 items-center gap-1.5">
                          {getChangeTypeIcon(file)}
                          <span
                            className="truncate font-mono text-xs text-muted-foreground"
                            title={getFileLabel(file)}
                          >
                            {getFileLabel(file)}
                          </span>
                        </div>
                        <button
                          onClick={() =>
                            setCollapsedFiles((prev) => ({
                              ...prev,
                              [fileKey]: false,
                            }))
                          }
                          className="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors"
                        >
                          <ChevronRight className="h-3.5 w-3.5" />
                          Expand
                        </button>
                      </div>
                    ) : (
                      <>
                        {collapsible && (
                          <button
                            onClick={() =>
                              setCollapsedFiles((prev) => ({
                                ...prev,
                                [fileKey]: true,
                              }))
                            }
                            className="absolute right-2 top-2 z-10 inline-flex items-center gap-1 rounded-md border bg-background/90 px-2 py-1 text-xs text-muted-foreground hover:bg-muted/70 hover:text-foreground transition-colors"
                          >
                            <ChevronDown className="h-3.5 w-3.5" />
                            Collapse
                          </button>
                        )}
                        <FileDiff fileDiff={file} options={DIFF_OPTIONS} className="block" />
                      </>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
