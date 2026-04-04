import { useState, useEffect, useCallback } from "react";
import {
  GetAudioDevices,
  InstallAPOOnDevice,
  UninstallAPOFromDevice,
  InstallAPOOnAllRender,
} from "../../wailsjs/go/main/App";

// ── Types ──────────────────────────────────────────────────────────────────────

interface AudioDevice {
  id: string;
  name: string;
  type: "render" | "capture";
  hasAPO: boolean;
  state: number; // 1=Active, 2=Disabled, 4=NotPresent, 8=Unplugged
  default: boolean;
}

// DeviceState constants
const STATE_ACTIVE = 1;
const STATE_DISABLED = 2;
const STATE_UNPLUGGED = 8;

// ── Sub-components ─────────────────────────────────────────────────────────────

function Statebadge({ state }: { state: number }) {
  if (state === STATE_ACTIVE)
    return (
      <span className="device-badge badge-active">
        <span className="badge-dot" />
        Active
      </span>
    );
  if (state === STATE_DISABLED)
    return <span className="device-badge badge-disabled">Disabled</span>;
  if (state === STATE_UNPLUGGED)
    return <span className="device-badge badge-unplugged">Unplugged</span>;
  return <span className="device-badge badge-disabled">Not Present</span>;
}

function DeviceIcon({ type }: { type: string }) {
  if (type === "capture") {
    // Microphone icon
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <rect x="9" y="2" width="6" height="11" rx="3" />
        <path d="M5 10a7 7 0 0 0 14 0" />
        <line x1="12" y1="19" x2="12" y2="22" />
        <line x1="8" y1="22" x2="16" y2="22" />
      </svg>
    );
  }
  // Speaker icon
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
      <path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
      <path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
    </svg>
  );
}

// ── Main Component ─────────────────────────────────────────────────────────────

interface AudioDevicesProps {
  onClose?: () => void;
}

export default function AudioDevices({ onClose }: AudioDevicesProps) {
  const [devices, setDevices] = useState<AudioDevice[]>([]);
  const [tab, setTab] = useState<"render" | "capture">("render");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null); // deviceId being toggled
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  // ── Data fetching ────────────────────────────────────────────────────────────

  const fetchDevices = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await GetAudioDevices();
      setDevices((result ?? []) as AudioDevice[]);
    } catch (e: any) {
      setError(e?.message ?? "Failed to load devices");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  // ── Actions ──────────────────────────────────────────────────────────────────

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 2800);
  };

  const toggleAPO = async (device: AudioDevice) => {
    setBusy(device.id);
    setError(null);
    try {
      if (device.hasAPO) {
        await UninstallAPOFromDevice(device.id, device.type);
        showToast(`Driver removed from "${device.name}"`);
      } else {
        await InstallAPOOnDevice(device.id, device.type);
        showToast(`Driver installed on "${device.name}"`);
      }
      await fetchDevices();
    } catch (e: any) {
      setError(e?.message ?? "Operation failed — run as Administrator");
    } finally {
      setBusy(null);
    }
  };

  const installAll = async () => {
    setBusy("__all__");
    setError(null);
    try {
      await InstallAPOOnAllRender();
      showToast("Driver installed on all active outputs");
      await fetchDevices();
    } catch (e: any) {
      setError(e?.message ?? "Operation failed — run as Administrator");
    } finally {
      setBusy(null);
    }
  };

  // ── Derived data ─────────────────────────────────────────────────────────────

  const filtered = devices.filter((d) => d.type === tab);
  const activeCount = filtered.filter((d) => d.state === STATE_ACTIVE).length;
  const apoCount = filtered.filter((d) => d.hasAPO).length;

  // ── Render ───────────────────────────────────────────────────────────────────

  return (
    <>
      <style>{styles}</style>

      <div className="ad-panel">
        {/* Header */}
        <div className="ad-header">
          <div className="ad-header-left">
            <div className="ad-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M9 18V5l12-2v13" />
                <circle cx="6" cy="18" r="3" />
                <circle cx="18" cy="16" r="3" />
              </svg>
            </div>
            <div>
              <h2 className="ad-title">Audio Devices</h2>
              <p className="ad-subtitle">APO Driver Configuration</p>
            </div>
          </div>
          {onClose && (
            <button className="ad-close" onClick={onClose}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          )}
        </div>

        {/* Tab switcher */}
        <div className="ad-tabs">
          <button
            className={`ad-tab ${tab === "render" ? "active" : ""}`}
            onClick={() => setTab("render")}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
              <path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
            </svg>
            Outputs
          </button>
          <button
            className={`ad-tab ${tab === "capture" ? "active" : ""}`}
            onClick={() => setTab("capture")}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="9" y="2" width="6" height="11" rx="3" />
              <path d="M5 10a7 7 0 0 0 14 0" />
              <line x1="12" y1="19" x2="12" y2="22" />
            </svg>
            Inputs
          </button>

          <div className="ad-tab-stats">
            <span className="stat-chip">{activeCount} active</span>
            <span className="stat-chip stat-apo">{apoCount} with APO</span>
          </div>
        </div>

        {/* Toolbar */}
        {tab === "render" && (
          <div className="ad-toolbar">
            <button
              className="ad-btn-all"
              onClick={installAll}
              disabled={busy !== null}
            >
              {busy === "__all__" ? (
                <span className="spinner" />
              ) : (
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <polyline points="16 16 12 12 8 16" />
                  <line x1="12" y1="12" x2="12" y2="21" />
                  <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3" />
                </svg>
              )}
              Install on All Active Outputs
            </button>

            <button className="ad-btn-refresh" onClick={fetchDevices} disabled={loading}>
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className={loading ? "spin" : ""}
              >
                <polyline points="23 4 23 10 17 10" />
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
              </svg>
            </button>
          </div>
        )}

        {/* Error banner */}
        {error && (
          <div className="ad-error">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
            {error}
          </div>
        )}

        {/* Device list */}
        <div className="ad-list">
          {loading ? (
            <div className="ad-empty">
              <div className="loading-dots">
                <span /><span /><span />
              </div>
              <p>Scanning devices…</p>
            </div>
          ) : filtered.length === 0 ? (
            <div className="ad-empty">
              <DeviceIcon type={tab} />
              <p>No {tab === "render" ? "output" : "input"} devices found</p>
            </div>
          ) : (
            filtered.map((device) => (
              <div
                key={device.id}
                className={`ad-device ${device.hasAPO ? "has-apo" : ""} ${
                  device.state !== STATE_ACTIVE ? "inactive" : ""
                }`}
              >
                {/* APO indicator strip */}
                {device.hasAPO && <div className="apo-strip" />}

                {/* Device icon */}
                <div className="device-icon-wrap">
                  <DeviceIcon type={device.type} />
                </div>

                {/* Device info */}
                <div className="device-info">
                  <span className="device-name">{device.name}</span>
                  <div className="device-meta">
                    <Statebadge state={device.state} />
                    {device.hasAPO && (
                      <span className="apo-badge">
                        <svg viewBox="0 0 24 24" fill="currentColor" width="10" height="10">
                          <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
                        </svg>
                        ViPER APO
                      </span>
                    )}
                  </div>
                </div>

                {/* Toggle button */}
                <button
                  className={`device-toggle ${device.hasAPO ? "toggle-remove" : "toggle-install"}`}
                  onClick={() => toggleAPO(device)}
                  disabled={busy !== null || device.state === 4}
                  title={device.hasAPO ? "Remove APO from this device" : "Install APO on this device"}
                >
                  {busy === device.id ? (
                    <span className="spinner" />
                  ) : device.hasAPO ? (
                    <>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6l-1 14H6L5 6" />
                        <path d="M10 11v6M14 11v6" />
                      </svg>
                      Remove
                    </>
                  ) : (
                    <>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <line x1="12" y1="5" x2="12" y2="19" />
                        <line x1="5" y1="12" x2="19" y2="12" />
                      </svg>
                      Install
                    </>
                  )}
                </button>
              </div>
            ))
          )}
        </div>

        {/* Toast notification */}
        {toast && <div className="ad-toast">{toast}</div>}
      </div>
    </>
  );
}

// ── Styles ─────────────────────────────────────────────────────────────────────

const styles = `
  .ad-panel {
    background: #141414;
    border: 1px solid #2a2a2a;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    font-family: 'DM Sans', 'Segoe UI', sans-serif;
    min-width: 340px;
    max-width: 420px;
  }

  /* ── Header ── */
  .ad-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 18px 12px;
    border-bottom: 1px solid #222;
  }

  .ad-header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .ad-icon {
    width: 36px;
    height: 36px;
    background: linear-gradient(135deg, #1e1e1e, #2a2a2a);
    border: 1px solid #333;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #e53935;
  }

  .ad-icon svg {
    width: 18px;
    height: 18px;
  }

  .ad-title {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
    color: #f0f0f0;
    letter-spacing: 0.02em;
  }

  .ad-subtitle {
    margin: 1px 0 0;
    font-size: 10px;
    color: #555;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  .ad-close {
    width: 28px;
    height: 28px;
    background: none;
    border: 1px solid #2a2a2a;
    border-radius: 6px;
    cursor: pointer;
    color: #555;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
    padding: 0;
  }

  .ad-close:hover { color: #f0f0f0; border-color: #444; background: #1e1e1e; }
  .ad-close svg { width: 14px; height: 14px; }

  /* ── Tabs ── */
  .ad-tabs {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 10px 14px 0;
    border-bottom: 1px solid #1e1e1e;
  }

  .ad-tab {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 12px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    cursor: pointer;
    color: #555;
    font-size: 11.5px;
    font-weight: 500;
    font-family: inherit;
    transition: all 0.15s;
    margin-bottom: -1px;
  }

  .ad-tab svg { width: 13px; height: 13px; }
  .ad-tab:hover { color: #aaa; }
  .ad-tab.active { color: #e53935; border-bottom-color: #e53935; }

  .ad-tab-stats {
    margin-left: auto;
    display: flex;
    gap: 6px;
    padding-bottom: 8px;
  }

  .stat-chip {
    font-size: 10px;
    padding: 2px 7px;
    border-radius: 20px;
    background: #1e1e1e;
    color: #555;
    border: 1px solid #2a2a2a;
  }

  .stat-apo {
    background: rgba(229, 57, 53, 0.08);
    color: #e53935;
    border-color: rgba(229, 57, 53, 0.2);
  }

  /* ── Toolbar ── */
  .ad-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 14px;
    border-bottom: 1px solid #1e1e1e;
  }

  .ad-btn-all {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    padding: 7px 12px;
    background: rgba(229, 57, 53, 0.1);
    border: 1px solid rgba(229, 57, 53, 0.25);
    border-radius: 7px;
    color: #e53935;
    font-size: 11.5px;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    transition: all 0.15s;
  }

  .ad-btn-all:hover:not(:disabled) {
    background: rgba(229, 57, 53, 0.18);
    border-color: rgba(229, 57, 53, 0.4);
  }

  .ad-btn-all:disabled { opacity: 0.45; cursor: not-allowed; }
  .ad-btn-all svg { width: 13px; height: 13px; }

  .ad-btn-refresh {
    width: 32px;
    height: 32px;
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 7px;
    color: #555;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
    padding: 0;
    flex-shrink: 0;
  }

  .ad-btn-refresh:hover { color: #aaa; border-color: #3a3a3a; }
  .ad-btn-refresh:disabled { opacity: 0.4; cursor: not-allowed; }
  .ad-btn-refresh svg { width: 14px; height: 14px; }

  /* ── Error ── */
  .ad-error {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 8px 14px 0;
    padding: 8px 12px;
    background: rgba(229, 57, 53, 0.08);
    border: 1px solid rgba(229, 57, 53, 0.2);
    border-radius: 7px;
    color: #e57373;
    font-size: 11px;
    line-height: 1.4;
  }

  .ad-error svg { width: 14px; height: 14px; flex-shrink: 0; }

  /* ── Device list ── */
  .ad-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 320px;
  }

  .ad-list::-webkit-scrollbar { width: 4px; }
  .ad-list::-webkit-scrollbar-track { background: transparent; }
  .ad-list::-webkit-scrollbar-thumb { background: #2a2a2a; border-radius: 2px; }

  .ad-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 32px;
    color: #444;
  }

  .ad-empty svg { width: 28px; height: 28px; }
  .ad-empty p { margin: 0; font-size: 12px; }

  /* ── Individual device row ── */
  .ad-device {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 11px;
    background: #181818;
    border: 1px solid #232323;
    border-radius: 8px;
    position: relative;
    overflow: hidden;
    transition: border-color 0.15s;
  }

  .ad-device:hover { border-color: #333; }
  .ad-device.has-apo { border-color: rgba(229, 57, 53, 0.2); background: rgba(229, 57, 53, 0.03); }
  .ad-device.inactive { opacity: 0.5; }

  /* Left glow strip for APO-enabled devices */
  .apo-strip {
    position: absolute;
    left: 0; top: 0; bottom: 0;
    width: 2px;
    background: linear-gradient(180deg, #e53935, #7b1fa2);
  }

  .device-icon-wrap {
    width: 30px;
    height: 30px;
    background: #1e1e1e;
    border: 1px solid #2a2a2a;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #555;
    flex-shrink: 0;
  }

  .ad-device.has-apo .device-icon-wrap { color: #e53935; border-color: rgba(229, 57, 53, 0.2); }
  .device-icon-wrap svg { width: 14px; height: 14px; }

  .device-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .device-name {
    font-size: 12px;
    font-weight: 500;
    color: #d0d0d0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .device-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }

  /* State badges */
  .device-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 10px;
  }

  .badge-active { background: rgba(76, 175, 80, 0.12); color: #66bb6a; }
  .badge-disabled { background: #1e1e1e; color: #555; border: 1px solid #2a2a2a; }
  .badge-unplugged { background: rgba(255, 152, 0, 0.1); color: #ffa726; }

  .badge-dot {
    width: 5px;
    height: 5px;
    background: currentColor;
    border-radius: 50%;
    animation: pulse-dot 2s ease-in-out infinite;
  }

  .apo-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 10px;
    background: rgba(229, 57, 53, 0.1);
    color: #e57373;
    border: 1px solid rgba(229, 57, 53, 0.15);
  }

  /* Toggle buttons */
  .device-toggle {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
    flex-shrink: 0;
    border: 1px solid;
  }

  .device-toggle svg { width: 11px; height: 11px; }

  .toggle-install {
    background: rgba(229, 57, 53, 0.1);
    border-color: rgba(229, 57, 53, 0.25);
    color: #e53935;
  }

  .toggle-install:hover:not(:disabled) {
    background: rgba(229, 57, 53, 0.2);
    border-color: rgba(229, 57, 53, 0.45);
  }

  .toggle-remove {
    background: #1e1e1e;
    border-color: #2a2a2a;
    color: #666;
  }

  .toggle-remove:hover:not(:disabled) {
    background: rgba(229, 57, 53, 0.06);
    border-color: rgba(229, 57, 53, 0.2);
    color: #e57373;
  }

  .device-toggle:disabled { opacity: 0.4; cursor: not-allowed; }

  /* ── Loading dots ── */
  .loading-dots {
    display: flex;
    gap: 5px;
  }

  .loading-dots span {
    width: 6px;
    height: 6px;
    background: #333;
    border-radius: 50%;
    animation: bounce 1.2s ease-in-out infinite;
  }

  .loading-dots span:nth-child(2) { animation-delay: 0.2s; }
  .loading-dots span:nth-child(3) { animation-delay: 0.4s; }

  /* ── Spinner ── */
  .spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 1.5px solid rgba(229, 57, 53, 0.25);
    border-top-color: #e53935;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  .spin { animation: spin 0.7s linear infinite; }

  /* ── Toast ── */
  .ad-toast {
    position: absolute;
    bottom: 12px;
    left: 50%;
    transform: translateX(-50%);
    background: #232323;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 8px 14px;
    font-size: 11.5px;
    color: #d0d0d0;
    white-space: nowrap;
    animation: toast-in 0.2s ease, toast-out 0.3s ease 2.5s forwards;
    pointer-events: none;
    box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  }

  /* ── Keyframes ── */
  @keyframes spin { to { transform: rotate(360deg); } }
  @keyframes bounce {
    0%, 80%, 100% { transform: scale(0.7); opacity: 0.4; }
    40% { transform: scale(1); opacity: 1; }
  }
  @keyframes pulse-dot {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
  @keyframes toast-in {
    from { opacity: 0; transform: translateX(-50%) translateY(6px); }
    to   { opacity: 1; transform: translateX(-50%) translateY(0); }
  }
  @keyframes toast-out {
    to { opacity: 0; transform: translateX(-50%) translateY(6px); }
  }
`;
