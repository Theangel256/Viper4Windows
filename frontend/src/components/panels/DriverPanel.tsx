import { useCallback, useEffect, useState } from "react";
import { ShieldCheck, ShieldAlert, RotateCw, Power, AlertTriangle } from "lucide-react";
import { useAudioStore } from "../../store/audioStore";
import { GetAudioDevices } from "../../wailsjs/go/app/App";
import { DSPButton } from "../ui/DSPButton";

/**
 * DriverPanel — system-level driver health & recovery.
 *
 * This is the missing piece behind "ningún efecto funciona" /
 * "no se reinicia el sistema de audio": InstallAPOOnDevice only ever
 * pointed a device's FxProperties at the ViPER CLSID — it never
 * registered that CLSID with Windows in the first place, and nothing
 * ever restarted audiodg.exe so a change would actually take effect.
 * "Reinstall + Recycle" below now does both for real (App.InstallDriver
 * on the Go side).
 *
 * Also surfaces "legacy residue": a device whose FxProperties still
 * points at some OTHER CLSID (an old, separately-uninstalled
 * ViPER4Windows that deregistered itself but never cleaned per-device
 * pointers) — exactly the "always shows installed but nothing plays"
 * symptom. Per-device attach/detach stays in the Audio Devices panel;
 * this view is about the driver as a whole.
 */

interface DeviceSummary {
  id: string;
  name: string;
  deviceType: string;
  hasAPO: boolean;
  legacyResidue: boolean;
  legacyClsid: string;
}

interface DriverPanelProps {
  onClose?: () => void;
  onManageDevices?: () => void;
}

export default function DriverPanel({ onClose, onManageDevices }: DriverPanelProps) {
  const {
    isDriverInstalled,
    isDriverAttached,
    driverVersion,
    driverArchitecture,
    driverDllPath,
    audioEngineRunning,
    isElevated,
    installDriver,
    restartAudioEngine,
  } = useAudioStore();

  const [devices, setDevices] = useState<DeviceSummary[]>([]);
  const [busy, setBusy] = useState<"install" | "restart" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refreshDevices = useCallback(async () => {
    try {
      const result = await GetAudioDevices();
      setDevices((result ?? []) as unknown as DeviceSummary[]);
    } catch {
      // Non-fatal here — the headline driver status still renders.
    }
  }, []);

  useEffect(() => {
    refreshDevices();
  }, [refreshDevices]);

  const legacyDevices = devices.filter((d) => d.legacyResidue);
  const healthy = isDriverInstalled && !legacyDevices.length;

  const handleInstall = async () => {
    setBusy("install");
    setError(null);
    try {
      await installDriver();
      await refreshDevices();
    } catch (e: any) {
      setError(e?.message ?? "Install failed — make sure you're running as Administrator");
    } finally {
      setBusy(null);
    }
  };

  const handleRestart = async () => {
    setBusy("restart");
    setError(null);
    try {
      await restartAudioEngine();
    } catch (e: any) {
      setError(e?.message ?? "Could not restart the audio engine");
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="rounded-[30px] border border-zinc-200 bg-white p-1 shadow-2xl dark:border-zinc-800 dark:bg-zinc-950">
      <div className="rounded-[26px] bg-zinc-50/60 p-6 dark:bg-zinc-900/40">
        {/* Header */}
        <div className="mb-5 flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div
              className={`flex h-11 w-11 items-center justify-center rounded-2xl ${
                healthy ? "bg-emerald-500/10 text-emerald-500" : "bg-amber-500/10 text-amber-500"
              }`}
            >
              {healthy ? <ShieldCheck size={20} /> : <ShieldAlert size={20} />}
            </div>
            <div>
              <h2 className="text-[15px] font-bold text-zinc-900 dark:text-zinc-50">
                {healthy ? "Driver is registered" : isDriverInstalled ? "Legacy residue detected" : "Driver not installed"}
              </h2>
              <p className="mt-0.5 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-400">
                APO Driver Health
              </p>
            </div>
          </div>
          {onClose && (
            <button
              onClick={onClose}
              className="rounded-xl border border-zinc-200 p-2 text-zinc-400 transition-colors hover:text-zinc-700 dark:border-zinc-800 dark:hover:text-zinc-200"
            >
              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth={2}>
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          )}
        </div>

        {!isElevated && (
          <div className="mb-4 flex items-center gap-2 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-[12px] text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-300">
            <AlertTriangle size={14} className="shrink-0" />
            Not running elevated — driver install/uninstall will fail. Restart as Administrator.
          </div>
        )}

        {legacyDevices.length > 0 && (
          <div className="mb-4 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-900/40 dark:bg-amber-950/20">
            <div className="flex items-center gap-2 text-[12px] font-semibold text-amber-800 dark:text-amber-300">
              <AlertTriangle size={14} />
              Old ViPER4Windows residue on {legacyDevices.length} device{legacyDevices.length > 1 ? "s" : ""}
            </div>
            <p className="mt-1 text-[11px] leading-relaxed text-amber-700/90 dark:text-amber-400/80">
              {legacyDevices[0].name} still points at CLSID {legacyDevices[0].legacyClsid} — an old install's
              configurator deregistered itself from Windows but never cleaned this endpoint's FxProperties. That's
              why it can look "installed" while no effect plays. "Reinstall + Recycle" below overwrites it.
            </p>
          </div>
        )}

        {error && (
          <div className="mb-4 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-[12px] text-rose-700 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-300">
            {error}
          </div>
        )}

        {/* Status rows */}
        <div className="mb-5 divide-y divide-zinc-200/70 overflow-hidden rounded-2xl border border-zinc-200/70 bg-white dark:divide-zinc-800/70 dark:border-zinc-800/70 dark:bg-zinc-900/60">
          <StatusRow label="Driver" value={isDriverInstalled ? "Installed" : "Not installed"} ok={isDriverInstalled} />
          <StatusRow label="Attached to outputs" value={isDriverAttached ? "Yes" : "No"} ok={isDriverAttached} />
          <StatusRow
            label="Audio engine"
            value={audioEngineRunning ? "Running" : "Stopped"}
            ok={audioEngineRunning}
          />
          <StatusRow label="Version" value={driverVersion || "—"} plain />
          <StatusRow label="Architecture" value={driverArchitecture || "—"} plain />
          <StatusRow label="DLL path" value={driverDllPath || "—"} plain mono />
        </div>

        {/* Actions */}
        <div className="flex flex-col gap-2 sm:flex-row">
          <DSPButton
            variant="danger"
            size="lg"
            className="flex-1 justify-center"
            onClick={handleInstall}
            disabled={busy !== null}
          >
            {busy === "install" ? (
              <RotateCw size={14} className="animate-spin" />
            ) : (
              <Power size={14} />
            )}
            Reinstall + Recycle
          </DSPButton>
          <DSPButton
            variant="ghost"
            size="lg"
            className="flex-1 justify-center border border-zinc-300 dark:border-zinc-700"
            onClick={handleRestart}
            disabled={busy !== null}
          >
            {busy === "restart" ? <RotateCw size={14} className="animate-spin" /> : <RotateCw size={14} />}
            Restart Audio Engine Only
          </DSPButton>
        </div>

        {onManageDevices && (
          <button
            onClick={onManageDevices}
            className="mt-3 w-full text-center text-[11px] font-semibold text-zinc-400 transition-colors hover:text-zinc-700 dark:hover:text-zinc-200"
          >
            Manage individual devices →
          </button>
        )}
      </div>
    </div>
  );
}

function StatusRow({
  label,
  value,
  ok,
  plain,
  mono,
}: {
  label: string;
  value: string;
  ok?: boolean;
  plain?: boolean;
  mono?: boolean;
}) {
  return (
    <div className="flex items-center justify-between px-4 py-3">
      <span className="text-[12px] text-zinc-500 dark:text-zinc-400">{label}</span>
      <span
        className={`flex items-center gap-1.5 text-[12px] font-semibold ${
          mono ? "truncate max-w-[220px] font-mono text-[10px]" : ""
        } ${
          plain
            ? "text-zinc-700 dark:text-zinc-300"
            : ok
              ? "text-emerald-600 dark:text-emerald-400"
              : "text-rose-600 dark:text-rose-400"
        }`}
        title={mono ? value : undefined}
      >
        {!plain && <span className={`h-1.5 w-1.5 rounded-full ${ok ? "bg-emerald-500" : "bg-rose-500"}`} />}
        {value}
      </span>
    </div>
  );
}
