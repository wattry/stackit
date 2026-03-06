import type { ReactNode } from "react";

// Language / framework brand icons (Simple Icons via react-icons)
import {
  SiGo,
  SiTypescript,
  SiJavascript,
  SiReact,
  SiCss,
  SiSass,
  SiHtml5,
  SiJson,
  SiYaml,
  SiToml,
  SiMarkdown,
  SiGnubash,
  SiPython,
  SiRust,
  SiRuby,
  SiDocker,
  SiSvg,
  SiEslint,
  SiGit,
} from "react-icons/si";
import {
  VscFile,
  VscFileMedia,
  VscGear,
  VscLock,
  VscDatabase,
} from "react-icons/vsc";

const iconClass = "h-3.5 w-3.5 shrink-0";

const extensionMap: Record<string, ReactNode> = {
  // Go
  ".go": <SiGo className={`${iconClass} text-cyan-500`} />,

  // TypeScript / JavaScript
  ".ts": <SiTypescript className={`${iconClass} text-blue-500`} />,
  ".tsx": <SiReact className={`${iconClass} text-blue-400`} />,
  ".js": <SiJavascript className={`${iconClass} text-yellow-500`} />,
  ".jsx": <SiReact className={`${iconClass} text-blue-400`} />,
  ".mjs": <SiJavascript className={`${iconClass} text-yellow-500`} />,
  ".cjs": <SiJavascript className={`${iconClass} text-yellow-500`} />,
  ".d.ts": <SiTypescript className={`${iconClass} text-blue-600 dark:text-blue-400`} />,

  // Web
  ".css": <SiCss className={`${iconClass} text-blue-500`} />,
  ".scss": <SiSass className={`${iconClass} text-pink-500`} />,
  ".html": <SiHtml5 className={`${iconClass} text-orange-500`} />,
  ".svg": <SiSvg className={`${iconClass} text-orange-400`} />,

  // Data / Config
  ".json": <SiJson className={`${iconClass} text-yellow-600 dark:text-yellow-400`} />,
  ".yaml": <SiYaml className={`${iconClass} text-red-400`} />,
  ".yml": <SiYaml className={`${iconClass} text-red-400`} />,
  ".toml": <SiToml className={`${iconClass} text-gray-500 dark:text-gray-400`} />,
  ".env": <VscGear className={`${iconClass} text-yellow-600 dark:text-yellow-400`} />,

  // Documentation
  ".md": <SiMarkdown className={`${iconClass} text-blue-400 dark:text-blue-300`} />,
  ".mdx": <SiMarkdown className={`${iconClass} text-blue-400 dark:text-blue-300`} />,
  ".txt": <VscFile className={`${iconClass} text-gray-400`} />,

  // Shell / Scripts
  ".sh": <SiGnubash className={`${iconClass} text-green-500`} />,
  ".bash": <SiGnubash className={`${iconClass} text-green-500`} />,
  ".zsh": <SiGnubash className={`${iconClass} text-green-500`} />,

  // Images
  ".png": <VscFileMedia className={`${iconClass} text-green-400`} />,
  ".jpg": <VscFileMedia className={`${iconClass} text-green-400`} />,
  ".jpeg": <VscFileMedia className={`${iconClass} text-green-400`} />,
  ".gif": <VscFileMedia className={`${iconClass} text-green-400`} />,
  ".webp": <VscFileMedia className={`${iconClass} text-green-400`} />,
  ".ico": <VscFileMedia className={`${iconClass} text-green-400`} />,

  // Languages
  ".rs": <SiRust className={`${iconClass} text-orange-600 dark:text-orange-400`} />,
  ".py": <SiPython className={`${iconClass} text-yellow-500`} />,
  ".rb": <SiRuby className={`${iconClass} text-red-500`} />,

  // Database
  ".sql": <VscDatabase className={`${iconClass} text-blue-300`} />,
};

const filenameMap: Record<string, ReactNode> = {
  "Dockerfile": <SiDocker className={`${iconClass} text-blue-500`} />,
  "Makefile": <VscGear className={`${iconClass} text-orange-500`} />,
  ".gitignore": <SiGit className={`${iconClass} text-orange-500`} />,
  ".eslintrc": <SiEslint className={`${iconClass} text-purple-500`} />,
  ".eslintrc.json": <SiEslint className={`${iconClass} text-purple-500`} />,
  "eslint.config.js": <SiEslint className={`${iconClass} text-purple-500`} />,
  "eslint.config.mjs": <SiEslint className={`${iconClass} text-purple-500`} />,
  "package.json": <SiJson className={`${iconClass} text-green-500`} />,
  "tsconfig.json": <SiTypescript className={`${iconClass} text-blue-500`} />,
  "go.mod": <SiGo className={`${iconClass} text-cyan-500`} />,
  "go.sum": <VscLock className={`${iconClass} text-cyan-400`} />,
  "pnpm-lock.yaml": <VscLock className={`${iconClass} text-orange-400`} />,
  "package-lock.json": <VscLock className={`${iconClass} text-orange-400`} />,
  "yarn.lock": <VscLock className={`${iconClass} text-blue-400`} />,
};

const defaultIcon = <VscFile className={`${iconClass} text-muted-foreground`} />;

export function getFileIcon(filename: string): ReactNode {
  // Check full filename first (e.g., Dockerfile, go.mod)
  const match = filenameMap[filename];
  if (match) return match;

  // Check compound extensions (e.g., .d.ts)
  const lowerName = filename.toLowerCase();
  if (lowerName.endsWith(".d.ts")) {
    return extensionMap[".d.ts"];
  }

  // Check simple extension
  const dotIndex = lowerName.lastIndexOf(".");
  if (dotIndex !== -1) {
    const ext = lowerName.slice(dotIndex);
    const extMatch = extensionMap[ext];
    if (extMatch) return extMatch;
  }

  return defaultIcon;
}
