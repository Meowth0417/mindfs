import React, { useState } from "react";
import {
  type GitHistoryItem,
  type GitHistoryPayload,
  type GitRemoteAction,
  type GitStatusItem,
  type GitStatusPayload,
} from "../services/git";
import { GitChangesPanel } from "./GitChangesPanel";

type RightSidebarTab = "sessions" | "git";

type RightSidebarProps = {
  children?: React.ReactNode;
  gitStatus?: GitStatusPayload | null;
  gitLoading?: boolean;
  gitHistory?: GitHistoryPayload | null;
  gitHistoryLoading?: boolean;
  gitHistoryLoadingMore?: boolean;
  showGitHistory?: boolean;
  gitHistoryExpandedCommits?: Record<string, boolean>;
  rootId?: string;
  rootName?: string;
  onSelectGitItem?: (item: GitStatusItem) => void;
  onGitCommit?: (message: string, amend: boolean) => Promise<void>;
  onGitCommitAndPush?: (message: string) => Promise<void>;
  onGitStageFile?: (path: string) => Promise<void>;
  onGitUnstageFile?: (path: string) => Promise<void>;
  onGitStageAll?: () => Promise<void>;
  onGitRemoteAction?: (action: GitRemoteAction) => Promise<void>;
  onToggleGitHistory?: () => void;
  onToggleGitHistoryCommit?: (hash: string) => void;
  onLoadMoreGitHistory?: () => void;
  onSelectGitHistoryFile?: (commit: GitHistoryItem, item: GitStatusItem) => void;
  showGit?: boolean;
};

export function RightSidebar({
  children,
  gitStatus,
  gitLoading,
  gitHistory,
  gitHistoryLoading,
  gitHistoryLoadingMore,
  showGitHistory = true,
  gitHistoryExpandedCommits = {},
  rootId,
  rootName,
  onSelectGitItem,
  onGitCommit,
  onGitCommitAndPush,
  onGitStageFile,
  onGitUnstageFile,
  onGitStageAll,
  onGitRemoteAction,
  onToggleGitHistory,
  onToggleGitHistoryCommit,
  onLoadMoreGitHistory,
  onSelectGitHistoryFile,
  showGit,
}: RightSidebarProps) {
  const [activeTab, setActiveTab] = useState<RightSidebarTab>("sessions");

  const tabs: { id: RightSidebarTab; label: string }[] = [
    { id: "sessions", label: "会话" },
    ...(showGit ? [{ id: "git" as RightSidebarTab, label: "Git" }] : []),
  ];

  return (
    <div style={{ flex: 1, minHeight: 0, display: "flex", flexDirection: "column" }}>
      {/* Tab header */}
      <div
        style={{
          height: "36px",
          padding: "0 8px",
          display: "flex",
          alignItems: "stretch",
          borderBottom: "1px solid var(--border-color)",
          background: "var(--mindfs-topbar-bg, transparent)",
          position: "sticky",
          top: 0,
          zIndex: 2,
          backdropFilter: "blur(8px)",
          boxSizing: "border-box",
          gap: "2px",
          flexShrink: 0,
        }}
      >
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            style={{
              border: "none",
              background: "transparent",
              padding: "0 10px",
              fontSize: "11px",
              fontWeight: 600,
              color: activeTab === tab.id ? "var(--text-primary)" : "var(--text-secondary)",
              borderBottom: activeTab === tab.id ? "2px solid var(--accent-color, #b45309)" : "2px solid transparent",
              cursor: "pointer",
              textTransform: "uppercase",
              letterSpacing: "0.04em",
              transition: "color 0.15s, border-color 0.15s",
              marginBottom: "-1px",
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, minHeight: 0, display: "flex", flexDirection: "column" }}>
        {activeTab === "sessions" ? (
          <div style={{ flex: 1, minHeight: 0, display: "flex", flexDirection: "column" }}>{children}</div>
        ) : activeTab === "git" ? (
          <GitChangesPanel
            rootId={rootId}
            rootName={rootName}
            status={gitStatus ?? null}
            loading={gitLoading}
            onSelectItem={onSelectGitItem}
            onCommit={onGitCommit}
            onCommitAndPush={onGitCommitAndPush}
            onStageFile={onGitStageFile}
            onUnstageFile={onGitUnstageFile}
            onStageAll={onGitStageAll}
            onRemoteAction={onGitRemoteAction}
            history={gitHistory ?? null}
            historyLoading={gitHistoryLoading}
            historyLoadingMore={gitHistoryLoadingMore}
            historyExpanded={showGitHistory}
            historyExpandedCommits={gitHistoryExpandedCommits}
            onToggleHistoryExpanded={onToggleGitHistory}
            onToggleHistoryCommit={onToggleGitHistoryCommit}
            onLoadMoreHistory={onLoadMoreGitHistory}
            onSelectHistoryFile={onSelectGitHistoryFile}
          />
        ) : null}
      </div>
    </div>
  );
}

