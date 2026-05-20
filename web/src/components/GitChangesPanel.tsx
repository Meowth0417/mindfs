import React, { useEffect, useRef, useState } from "react";
import {
  type GitHistoryItem,
  type GitHistoryPayload,
  type GitRemoteAction,
  type GitStatusItem,
  type GitStatusPayload,
} from "../services/git";
import { GitHistoryPanel } from "./GitHistoryPanel";

type GitChangesPanelProps = {
  rootId?: string;
  rootName?: string;
  status: GitStatusPayload | null;
  loading?: boolean;
  onSelectItem?: (item: GitStatusItem) => void;
  onCommit?: (message: string, amend: boolean) => Promise<void>;
  onCommitAndPush?: (message: string) => Promise<void>;
  onStageFile?: (path: string) => Promise<void>;
  onUnstageFile?: (path: string) => Promise<void>;
  onStageAll?: () => Promise<void>;
  onRemoteAction?: (action: GitRemoteAction) => Promise<void>;
  history?: GitHistoryPayload | null;
  historyLoading?: boolean;
  historyLoadingMore?: boolean;
  historyExpanded?: boolean;
  historyExpandedCommits?: Record<string, boolean>;
  onToggleHistoryExpanded?: () => void;
  onToggleHistoryCommit?: (hash: string) => void;
  onLoadMoreHistory?: () => void;
  onSelectHistoryFile?: (commit: GitHistoryItem, item: GitStatusItem) => void;
};

function renderStatusColor(status: string): string {
  switch (status) {
    case "A":
      return "#15803d";
    case "D":
      return "#b91c1c";
    case "R":
      return "#1d4ed8";
    case "??":
      return "#7c3aed";
    default:
      return "#b45309";
  }
}

function renderStatusLabel(status: string): string {
  return status === "??" ? "U" : status;
}

function renderLineStat(value: number, prefix: "+" | "-"): React.ReactNode {
  const color = prefix === "+" ? "#15803d" : "#b91c1c";
  return (
    <span style={{ color, fontVariantNumeric: "tabular-nums" }}>
      {prefix}
      {value}
    </span>
  );
}

function IconChevron({ expanded }: { expanded: boolean }) {
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
      style={{
        flexShrink: 0,
        transform: expanded ? "rotate(0deg)" : "rotate(-90deg)",
        transition: "transform 0.15s ease",
      }}
    >
      <path
        fillRule="evenodd"
        d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.17l3.71-3.94a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06"
        clipRule="evenodd"
      />
    </svg>
  );
}

function IconRepo() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M6 4.75A2.75 2.75 0 0 1 8.75 2h6.5A2.75 2.75 0 0 1 18 4.75v14.5A2.75 2.75 0 0 1 15.25 22h-6.5A2.75 2.75 0 0 1 6 19.25V4.75Z"
        stroke="currentColor"
        strokeWidth="1.7"
      />
      <path d="M9 18h6" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}

function IconBranch() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M6 3v12m0 0a3 3 0 1 0 0 6a3 3 0 0 0 0-6Zm12-9a3 3 0 1 0 0 6a3 3 0 0 0 0-6Zm0 0v3a6 6 0 0 1-6 6H9"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function IconDots() {
  return (
    <svg width="14" height="14" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
      <path d="M4 10a1.5 1.5 0 1 1 3 0a1.5 1.5 0 0 1-3 0Zm4.5 0a1.5 1.5 0 1 1 3 0a1.5 1.5 0 0 1-3 0ZM13 10a1.5 1.5 0 1 1 3 0a1.5 1.5 0 0 1-3 0Z" />
    </svg>
  );
}

function IconCaret({ open }: { open: boolean }) {
  return (
    <svg
      width="11"
      height="11"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
      style={{
        transform: open ? "rotate(180deg)" : "rotate(0deg)",
        transition: "transform 0.15s ease",
      }}
    >
      <path
        fillRule="evenodd"
        d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.17l3.71-3.94a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06"
        clipRule="evenodd"
      />
    </svg>
  );
}

function IconPlus() {
  return (
    <svg width="12" height="12" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
      <path d="M10.75 4.75a.75.75 0 0 0-1.5 0v4.5h-4.5a.75.75 0 0 0 0 1.5h4.5v4.5a.75.75 0 0 0 1.5 0v-4.5h4.5a.75.75 0 0 0 0-1.5h-4.5v-4.5Z" />
    </svg>
  );
}

function IconMinus() {
  return (
    <svg width="12" height="12" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M4 10a.75.75 0 0 1 .75-.75h10.5a.75.75 0 0 1 0 1.5H4.75A.75.75 0 0 1 4 10Z"
        clipRule="evenodd"
      />
    </svg>
  );
}

type FileRowProps = {
  item: GitStatusItem;
  actionIcon: React.ReactNode;
  actionTitle: string;
  onAction?: (path: string) => Promise<void>;
  onSelectItem?: (item: GitStatusItem) => void;
};

function FileRow({
  item,
  actionIcon,
  actionTitle,
  onAction,
  onSelectItem,
}: FileRowProps) {
  const [busy, setBusy] = useState(false);

  const handleAction = async (event: React.MouseEvent) => {
    event.stopPropagation();
    if (!onAction || busy) {
      return;
    }
    setBusy(true);
    try {
      await onAction(item.path);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      onClick={() => {
        if (!item.is_dir) {
          onSelectItem?.(item);
        }
      }}
      style={{
        display: "flex",
        alignItems: "center",
        gap: "6px",
        width: "100%",
        padding: "4px 6px 4px 18px",
        borderRadius: "6px",
        cursor: item.is_dir ? "default" : "pointer",
        opacity: item.is_dir ? 0.7 : 1,
        boxSizing: "border-box",
      }}
      onMouseEnter={(event) => {
        event.currentTarget.style.background = "var(--selection-bg, rgba(59,130,246,0.09))";
      }}
      onMouseLeave={(event) => {
        event.currentTarget.style.background = "transparent";
      }}
    >
      <span
        style={{
          width: "18px",
          color: renderStatusColor(item.status),
          fontSize: "11px",
          fontWeight: 700,
          flexShrink: 0,
        }}
      >
        {renderStatusLabel(item.status)}
      </span>
      <span
        style={{
          flex: 1,
          minWidth: 0,
          fontSize: "12px",
          color: "var(--text-primary)",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {item.display_path || item.path}
      </span>
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: "6px",
          fontSize: "10px",
          color: "var(--text-secondary)",
          flexShrink: 0,
        }}
      >
        {renderLineStat(item.additions, "+")}
        {renderLineStat(item.deletions, "-")}
      </span>
      {onAction ? (
        <button
          type="button"
          title={actionTitle}
          disabled={busy}
          onClick={handleAction}
          style={{
            width: "18px",
            height: "18px",
            border: "none",
            borderRadius: "4px",
            background: "transparent",
            color: "var(--text-secondary)",
            display: "inline-flex",
            alignItems: "center",
            justifyContent: "center",
            cursor: busy ? "not-allowed" : "pointer",
            flexShrink: 0,
            padding: 0,
            opacity: busy ? 0.5 : 1,
          }}
          onMouseEnter={(event) => {
            event.currentTarget.style.background = "var(--selection-bg, rgba(59,130,246,0.12))";
            event.currentTarget.style.color = "var(--text-primary)";
          }}
          onMouseLeave={(event) => {
            event.currentTarget.style.background = "transparent";
            event.currentTarget.style.color = "var(--text-secondary)";
          }}
        >
          {busy ? "·" : actionIcon}
        </button>
      ) : null}
    </div>
  );
}

type ChangeGroupProps = {
  title: string;
  count: number;
  items: GitStatusItem[];
  actionIcon: React.ReactNode;
  actionTitle: string;
  headerAction?: React.ReactNode;
  onFileAction?: (path: string) => Promise<void>;
  onSelectItem?: (item: GitStatusItem) => void;
};

function ChangeGroup({
  title,
  count,
  items,
  actionIcon,
  actionTitle,
  headerAction,
  onFileAction,
  onSelectItem,
}: ChangeGroupProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "2px" }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "6px",
          padding: "4px 6px 4px 18px",
        }}
      >
        <span
          style={{
            flex: 1,
            minWidth: 0,
            fontSize: "11px",
            color: "var(--text-secondary)",
            fontWeight: 600,
          }}
        >
          {title}
        </span>
        <span
          style={{
            fontSize: "10px",
            color: "var(--text-secondary)",
            fontVariantNumeric: "tabular-nums",
          }}
        >
          {count}
        </span>
        {headerAction}
      </div>
      <div style={{ display: "flex", flexDirection: "column" }}>
        {items.map((item) => (
          <FileRow
            key={`${item.area}:${item.status}:${item.path}`}
            item={item}
            actionIcon={actionIcon}
            actionTitle={actionTitle}
            onAction={onFileAction}
            onSelectItem={onSelectItem}
          />
        ))}
      </div>
    </div>
  );
}

type SectionHeaderProps = {
  title: string;
  expanded: boolean;
  count?: number;
  onToggle: () => void;
};

function SectionHeader({ title, expanded, count, onToggle }: SectionHeaderProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      style={{
        width: "100%",
        border: "none",
        background: "transparent",
        padding: "4px 6px",
        display: "flex",
        alignItems: "center",
        gap: "6px",
        cursor: "pointer",
        textAlign: "left",
      }}
    >
      <span style={{ color: "var(--text-secondary)" }}>
        <IconChevron expanded={expanded} />
      </span>
      <span
        style={{
          flex: 1,
          minWidth: 0,
          fontSize: "12px",
          fontWeight: 700,
          color: "var(--text-secondary)",
          textTransform: "uppercase",
          letterSpacing: "0.04em",
        }}
      >
        {title}
      </span>
      {typeof count === "number" ? (
        <span
          style={{
            fontSize: "11px",
            color: "var(--text-secondary)",
            fontVariantNumeric: "tabular-nums",
          }}
        >
          {count}
        </span>
      ) : null}
    </button>
  );
}

const remoteActionLabels: Record<GitRemoteAction, string> = {
  pull: "Pull",
  push: "Push",
  fetch: "Fetch",
};

export function GitChangesPanel({
  rootId,
  rootName,
  status,
  loading = false,
  onSelectItem,
  onCommit,
  onCommitAndPush,
  onStageFile,
  onUnstageFile,
  onStageAll,
  onRemoteAction,
  history,
  historyLoading = false,
  historyLoadingMore = false,
  historyExpanded = true,
  historyExpandedCommits = {},
  onToggleHistoryExpanded,
  onToggleHistoryCommit,
  onLoadMoreHistory,
  onSelectHistoryFile,
}: GitChangesPanelProps) {
  const [message, setMessage] = useState("");
  const [committing, setCommitting] = useState(false);
  const [commitError, setCommitError] = useState("");
  const [commitMenuOpen, setCommitMenuOpen] = useState(false);
  const [repositoryExpanded, setRepositoryExpanded] = useState(true);
  const [changesExpanded, setChangesExpanded] = useState(true);
  const [remoteMenuOpen, setRemoteMenuOpen] = useState(false);
  const [remoteBusy, setRemoteBusy] = useState<GitRemoteAction | "">("");
  const commitMenuRef = useRef<HTMLDivElement | null>(null);
  const remoteMenuRef = useRef<HTMLDivElement | null>(null);

  const branch = status?.branch || "";
  const allItems = status?.items || [];
  const stagedItems = allItems.filter((item) => item.area === "staged");
  const unstagedItems = allItems.filter((item) => item.area === "unstaged" || !item.area);
  const hasStagedChanges = stagedItems.length > 0;
  const historyItems = history?.items || [];
  const repoLabel =
    rootName?.trim()
    || rootId?.split(/[\\/]/).filter(Boolean).pop()
    || rootId
    || "Repository";

  useEffect(() => {
    if (!commitMenuOpen) {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      if (!commitMenuRef.current?.contains(event.target as Node)) {
        setCommitMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [commitMenuOpen]);

  useEffect(() => {
    if (!remoteMenuOpen) {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      if (!remoteMenuRef.current?.contains(event.target as Node)) {
        setRemoteMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [remoteMenuOpen]);

  const handleCommit = async (amend: boolean) => {
    if (!rootId || !onCommit) {
      return;
    }
    if (!amend && (!hasStagedChanges || !message.trim())) {
      return;
    }
    setCommitting(true);
    setCommitError("");
    setCommitMenuOpen(false);
    try {
      await onCommit(message, amend);
      setMessage("");
    } catch (err: any) {
      setCommitError(typeof err?.message === "string" ? err.message : String(err));
    } finally {
      setCommitting(false);
    }
  };

  const handleCommitAndPush = async () => {
    if (!rootId || !onCommitAndPush || !hasStagedChanges || !message.trim()) {
      return;
    }
    setCommitting(true);
    setCommitError("");
    setCommitMenuOpen(false);
    try {
      await onCommitAndPush(message);
      setMessage("");
    } catch (err: any) {
      setCommitError(typeof err?.message === "string" ? err.message : String(err));
    } finally {
      setCommitting(false);
    }
  };

  const handleRemoteAction = async (action: GitRemoteAction) => {
    if (!onRemoteAction || remoteBusy) {
      return;
    }
    setRemoteBusy(action);
    try {
      await onRemoteAction(action);
      setRemoteMenuOpen(false);
    } finally {
      setRemoteBusy("");
    }
  };

  const canCommit = !!rootId && !!onCommit && !committing && hasStagedChanges;
  const canCommitAndPush = !!rootId && !!onCommitAndPush && !committing && hasStagedChanges;
  const canSubmitCommit = canCommit && !!message.trim();
  const canSubmitCommitAndPush = canCommitAndPush && !!message.trim();

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        height: "100%",
        minHeight: 0,
        overflowY: "auto",
        padding: "8px 4px 14px",
        boxSizing: "border-box",
      }}
    >
      <SectionHeader
        title="Repositories"
        expanded={repositoryExpanded}
        count={1}
        onToggle={() => setRepositoryExpanded((value) => !value)}
      />
      {repositoryExpanded ? (
        <div style={{ padding: "0 6px 8px 18px" }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px",
              minHeight: "34px",
              borderRadius: "8px",
              padding: "4px 6px",
            }}
          >
            <span
              style={{
                width: "18px",
                height: "18px",
                display: "inline-flex",
                alignItems: "center",
                justifyContent: "center",
                color: "var(--text-secondary)",
                flexShrink: 0,
              }}
            >
              <IconRepo />
            </span>
            <span
              style={{
                flex: 1,
                minWidth: 0,
                fontSize: "13px",
                color: "var(--text-primary)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
            >
              {repoLabel}
            </span>
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: "4px",
                minWidth: 0,
                maxWidth: "45%",
                padding: "3px 8px",
                borderRadius: "999px",
                background: "var(--selection-bg, rgba(59,130,246,0.08))",
                color: "var(--text-primary)",
                fontSize: "12px",
                flexShrink: 1,
              }}
              title={branch || "detached HEAD"}
            >
              <span style={{ color: "var(--text-secondary)", flexShrink: 0 }}>
                <IconBranch />
              </span>
              <span
                style={{
                  minWidth: 0,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {branch || "detached HEAD"}
              </span>
            </span>
            <div ref={remoteMenuRef} style={{ position: "relative", flexShrink: 0 }}>
              <button
                type="button"
                aria-label="Repository actions"
                disabled={!onRemoteAction || !!remoteBusy}
                onClick={() => setRemoteMenuOpen((value) => !value)}
                style={{
                  width: "28px",
                  height: "28px",
                  border: "none",
                  borderRadius: "6px",
                  background: remoteMenuOpen
                    ? "var(--selection-bg, rgba(59,130,246,0.12))"
                    : "transparent",
                  color: "var(--text-secondary)",
                  display: "inline-flex",
                  alignItems: "center",
                  justifyContent: "center",
                  cursor: !onRemoteAction || remoteBusy ? "not-allowed" : "pointer",
                  opacity: !onRemoteAction || remoteBusy ? 0.5 : 1,
                  padding: 0,
                }}
              >
                <IconDots />
              </button>
              {remoteMenuOpen ? (
                <div
                  style={{
                    position: "absolute",
                    top: "calc(100% + 6px)",
                    right: 0,
                    minWidth: "150px",
                    background: "var(--menu-bg, var(--content-bg, #fff))",
                    border: "1px solid var(--border-color)",
                    borderRadius: "8px",
                    boxShadow: "0 10px 28px rgba(0,0,0,0.12)",
                    zIndex: 40,
                    padding: "4px",
                  }}
                >
                  {(["pull", "push", "fetch"] as GitRemoteAction[]).map((action) => {
                    const busy = remoteBusy === action;
                    return (
                      <button
                        key={action}
                        type="button"
                        disabled={!!remoteBusy}
                        onClick={() => {
                          void handleRemoteAction(action);
                        }}
                        style={{
                          width: "100%",
                          border: "none",
                          background: "transparent",
                          color: "var(--text-primary)",
                          borderRadius: "6px",
                          padding: "7px 10px",
                          fontSize: "12px",
                          textAlign: "left",
                          cursor: remoteBusy ? "not-allowed" : "pointer",
                          opacity: remoteBusy && !busy ? 0.5 : 1,
                        }}
                        onMouseEnter={(event) => {
                          event.currentTarget.style.background = "var(--selection-bg, rgba(59,130,246,0.1))";
                        }}
                        onMouseLeave={(event) => {
                          event.currentTarget.style.background = "transparent";
                        }}
                      >
                        {busy ? `${remoteActionLabels[action]}...` : remoteActionLabels[action]}
                      </button>
                    );
                  })}
                </div>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}

      <SectionHeader
        title="Changes"
        expanded={changesExpanded}
        count={allItems.length}
        onToggle={() => setChangesExpanded((value) => !value)}
      />
      {changesExpanded ? (
        <div style={{ display: "flex", flexDirection: "column", gap: "6px", paddingBottom: "8px" }}>
          <div style={{ padding: "0 6px 0 18px" }}>
            <textarea
              value={message}
              onChange={(event) => setMessage(event.target.value)}
              onKeyDown={(event) => {
                if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
                  event.preventDefault();
                  void handleCommit(false);
                }
              }}
              placeholder={
                branch
                  ? `Message (Ctrl+Enter to commit on "${branch}")`
                  : "Message (Ctrl+Enter to commit)"
              }
              rows={2}
              style={{
                width: "100%",
                boxSizing: "border-box",
                resize: "none",
                border: "1px solid var(--border-color)",
                borderRadius: "6px",
                padding: "7px 8px",
                fontSize: "12px",
                fontFamily: "inherit",
                background: "var(--input-bg, var(--content-bg))",
                color: "var(--text-primary)",
                outline: "none",
              }}
            />
          </div>
          <div style={{ padding: "0 6px 0 18px", display: "flex", gap: "2px" }}>
            <button
              type="button"
              disabled={!canSubmitCommit}
              onClick={() => {
                void handleCommit(false);
              }}
              style={{
                flex: 1,
                height: "28px",
                border: "none",
                borderRadius: "6px 0 0 6px",
                background: "var(--accent-color, #b45309)",
                color: "#fff",
                fontWeight: 600,
                fontSize: "12px",
                cursor: canSubmitCommit ? "pointer" : "not-allowed",
                opacity: canSubmitCommit ? 1 : 0.5,
              }}
            >
              {committing ? "Committing..." : "Commit"}
            </button>
            <div ref={commitMenuRef} style={{ position: "relative" }}>
              <button
                type="button"
                disabled={!canCommit}
                onClick={() => setCommitMenuOpen((value) => !value)}
                style={{
                  width: "26px",
                  height: "28px",
                  border: "none",
                  borderLeft: "1px solid rgba(255,255,255,0.25)",
                  borderRadius: "0 6px 6px 0",
                  background: "var(--accent-color, #b45309)",
                  color: "#fff",
                  cursor: canCommit ? "pointer" : "not-allowed",
                  opacity: canCommit ? 1 : 0.5,
                  display: "inline-flex",
                  alignItems: "center",
                  justifyContent: "center",
                  padding: 0,
                }}
              >
                <IconCaret open={commitMenuOpen} />
              </button>
              {commitMenuOpen ? (
                <div
                  style={{
                    position: "absolute",
                    top: "calc(100% + 4px)",
                    right: 0,
                    minWidth: "160px",
                    background: "var(--menu-bg, var(--content-bg, #fff))",
                    border: "1px solid var(--border-color)",
                    borderRadius: "8px",
                    boxShadow: "0 10px 28px rgba(0,0,0,0.12)",
                    zIndex: 40,
                    padding: "4px",
                  }}
                >
                  {[
                    { label: "Commit", amend: false, needMessage: true },
                    { label: "Commit & Push", commitAndPush: true, needMessage: true },
                    { label: "Amend Commit", amend: true, needMessage: false },
                  ].map(({ label, amend, needMessage, commitAndPush }) => (
                    <button
                      key={label}
                      type="button"
                      disabled={
                        committing
                        || (commitAndPush ? !canSubmitCommitAndPush : false)
                        || (!commitAndPush && amend !== true && !canSubmitCommit)
                        || (needMessage && !message.trim())
                      }
                      onClick={() => {
                        if (commitAndPush) {
                          void handleCommitAndPush();
                          return;
                        }
                        void handleCommit(amend ?? false);
                      }}
                      style={{
                        width: "100%",
                        border: "none",
                        background: "transparent",
                        color: "var(--text-primary)",
                        borderRadius: "6px",
                        padding: "7px 10px",
                        fontSize: "12px",
                        textAlign: "left",
                        cursor:
                          committing
                          || (commitAndPush ? !canSubmitCommitAndPush : false)
                          || (!commitAndPush && amend !== true && !canSubmitCommit)
                          || (needMessage && !message.trim())
                            ? "not-allowed"
                            : "pointer",
                        opacity:
                          committing
                          || (commitAndPush ? !canSubmitCommitAndPush : false)
                          || (!commitAndPush && amend !== true && !canSubmitCommit)
                          || (needMessage && !message.trim())
                            ? 0.5
                            : 1,
                      }}
                      onMouseEnter={(event) => {
                        event.currentTarget.style.background = "var(--selection-bg, rgba(59,130,246,0.1))";
                      }}
                      onMouseLeave={(event) => {
                        event.currentTarget.style.background = "transparent";
                      }}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          </div>
          {commitError ? (
            <div
              style={{
                margin: "0 6px 0 18px",
                padding: "6px 8px",
                borderRadius: "6px",
                background: "rgba(185,28,28,0.08)",
                border: "1px solid rgba(185,28,28,0.2)",
                fontSize: "11px",
                color: "#b91c1c",
              }}
            >
              {commitError}
            </div>
          ) : null}
          {loading ? (
            <div style={{ padding: "2px 6px 0 18px", fontSize: "12px", color: "var(--text-secondary)" }}>
              正在加载 git 变更...
            </div>
          ) : allItems.length === 0 ? (
            <div style={{ padding: "2px 6px 0 18px", fontSize: "12px", color: "var(--text-secondary)" }}>
              无未提交变更
            </div>
          ) : (
            <>
              <ChangeGroup
                title="Staged Changes"
                count={stagedItems.length}
                items={stagedItems}
                actionIcon={<IconMinus />}
                actionTitle="Unstage change"
                onFileAction={onUnstageFile}
                onSelectItem={onSelectItem}
              />
              <ChangeGroup
                title="Changes"
                count={unstagedItems.length}
                items={unstagedItems}
                actionIcon={<IconPlus />}
                actionTitle="Stage change"
                onFileAction={onStageFile}
                onSelectItem={onSelectItem}
                headerAction={
                  onStageAll && unstagedItems.length > 0 ? (
                    <button
                      type="button"
                      title="Stage all changes"
                      onClick={() => {
                        void onStageAll();
                      }}
                      style={{
                        width: "18px",
                        height: "18px",
                        border: "none",
                        borderRadius: "4px",
                        background: "transparent",
                        color: "var(--text-secondary)",
                        display: "inline-flex",
                        alignItems: "center",
                        justifyContent: "center",
                        cursor: "pointer",
                        padding: 0,
                        flexShrink: 0,
                      }}
                      onMouseEnter={(event) => {
                        event.currentTarget.style.background = "var(--selection-bg, rgba(59,130,246,0.12))";
                        event.currentTarget.style.color = "var(--text-primary)";
                      }}
                      onMouseLeave={(event) => {
                        event.currentTarget.style.background = "transparent";
                        event.currentTarget.style.color = "var(--text-secondary)";
                      }}
                    >
                      <IconPlus />
                    </button>
                  ) : undefined
                }
              />
            </>
          )}
        </div>
      ) : null}

      {rootId ? (
        <>
          <SectionHeader
            title="Timeline"
            expanded={historyExpanded}
            count={historyLoading ? undefined : historyItems.length}
            onToggle={() => onToggleHistoryExpanded?.()}
          />
          {historyExpanded ? (
            <div style={{ padding: "0 6px 0 12px" }}>
              {historyLoading || historyItems.length > 0 ? (
                <GitHistoryPanel
                  rootId={rootId}
                  items={historyItems}
                  loading={historyLoading}
                  loadingMore={historyLoadingMore}
                  hasMore={history?.has_more === true}
                  expandedCommits={historyExpandedCommits}
                  onToggleCommit={onToggleHistoryCommit}
                  onLoadMore={onLoadMoreHistory}
                  onSelectFile={onSelectHistoryFile}
                />
              ) : (
                <div style={{ padding: "2px 6px 0 12px", fontSize: "12px", color: "var(--text-secondary)" }}>
                  暂无历史提交
                </div>
              )}
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
