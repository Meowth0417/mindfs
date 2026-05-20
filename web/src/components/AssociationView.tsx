import React, { useState } from "react";

export type RelatedFile = {
  path: string;
  name: string;
  source_session?: string;
  session_name?: string;
  created_at?: string;
};

type AssociationViewProps = {
  title?: string;
  files: RelatedFile[];
  onFileClick?: (path: string) => void;
  onSessionClick?: (sessionKey: string) => void;
};

export function AssociationView({
  title = "关联文件",
  files,
  onFileClick,
  onSessionClick,
}: AssociationViewProps) {
  const [filter, setFilter] = useState("");

  const filteredFiles = files.filter(
    (f) =>
      f.name.toLowerCase().includes(filter.toLowerCase()) ||
      f.path.toLowerCase().includes(filter.toLowerCase())
  );

  return (
    <div style={{ padding: "24px", display: "flex", flexDirection: "column", gap: "20px" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <h2 style={{ margin: 0, fontSize: "18px", fontWeight: 600 }}>{title}</h2>
        <div style={{ fontSize: "13px", color: "var(--text-secondary)" }}>
          共 {files.length} 个关联项
        </div>
      </div>

      {/* 简单的搜索/过滤 */}
      <input
        type="text"
        placeholder="搜索关联文件..."
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        style={{
          padding: "8px 12px",
          borderRadius: "8px",
          border: "1px solid var(--border-color)",
          fontSize: "13px",
          width: "100%",
          maxWidth: "400px",
          outline: "none",
        }}
      />

      <div style={{ display: "flex", flexDirection: "column", gap: "8px" }}>
        {filteredFiles.length > 0 ? (
          filteredFiles.map((file, i) => (
            <div
              key={i}
              style={{
                display: "flex",
                alignItems: "center",
                gap: "12px",
                padding: "12px",
                background: "rgba(255,255,255,0.7)",
                border: "1px solid var(--border-color)",
                borderRadius: "10px",
                transition: "transform 0.1s, box-shadow 0.1s",
              }}
            >
              <span style={{ fontSize: "20px" }}>📄</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  onClick={() => onFileClick?.(file.path)}
                  style={{
                    fontSize: "14px",
                    fontWeight: 500,
                    cursor: "pointer",
                    color: "var(--accent-color)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {file.name}
                </div>
                <div
                  style={{
                    fontSize: "12px",
                    color: "var(--text-secondary)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {file.path}
                </div>
              </div>

              {file.source_session && (
                <button
                  onClick={() => onSessionClick?.(file.source_session!)}
                  style={{
                    padding: "4px 10px",
                    borderRadius: "6px",
                    border: "1px solid rgba(59,130,246,0.2)",
                    background: "rgba(59,130,246,0.05)",
                    color: "#3b82f6",
                    fontSize: "11px",
                    cursor: "pointer",
                  }}
                >
                  来源: {file.session_name || "查看会话"}
                </button>
              )}
            </div>
          ))
        ) : (
          <div style={{ padding: "40px", textAlign: "center", color: "var(--text-secondary)" }}>
            未找到匹配的关联文件
          </div>
        )}
      </div>
    </div>
  );
}
