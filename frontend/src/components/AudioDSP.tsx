import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { motion } from "framer-motion";
import { Activity, SlidersHorizontal, Speaker } from "lucide-react";

import { MasterControls } from "./panels/MasterControls";
import { XBass } from "./panels/XBass";
import { XClarity } from "./panels/XClarity";
import { Surround3D } from "./panels/Surround3D";
import { ReverbSidebar } from "./panels/ReverbSidebar";
import AudioDevices from "./panels/AudioDevices";
import { useAudioStore } from "../store/audioStore";
import { Sidebar } from "./Sidebar";
import { Equalizer } from "./Equalizer";
import { StatusHeader } from "./ui/StatusHeader";
import { EventsOn } from "../wailsjs/runtime/runtime";

function OverlayShell({
  size = "wide",
  onClose,
  children,
}: {
  size?: "wide" | "medium";
  onClose: () => void;
  children: React.ReactNode;
}) {
  if (typeof document === "undefined") {
    return null;
  }

  const shellSize =
    size === "medium"
      ? "max-w-[1040px] 2xl:max-w-[1120px]"
      : "max-w-[1380px] 2xl:max-w-[1480px]";

  return createPortal(
    <div className="fixed inset-0 z-[120] flex items-center justify-center p-3 sm:p-4 lg:p-6">
      <div className="absolute inset-0 bg-zinc-950/70 backdrop-blur-sm" onClick={onClose} />
      <div className={`relative w-full ${shellSize}`}>
        <div className="max-h-[calc(100vh-1.5rem)] overflow-y-auto custom-scrollbar rounded-[30px] p-1 sm:max-h-[calc(100vh-2.5rem)] sm:p-2">
          {children}
        </div>
      </div>
    </div>,
    document.body,
  );
}

function UtilityLauncher({
  icon,
  title,
  subtitle,
  accent,
  onClick,
}: {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  accent: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="group flex w-full items-center justify-between rounded-[26px] border border-zinc-200 dark:border-zinc-800 bg-white/95 dark:bg-zinc-900 px-5 py-4 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-zinc-300 dark:hover:border-zinc-700"
    >
      <div className="flex items-center gap-4">
        <div className={`flex h-11 w-11 items-center justify-center rounded-2xl ${accent}`}>
          {icon}
        </div>
        <div>
          <div className="text-[15px] font-semibold text-zinc-900 dark:text-zinc-50">{title}</div>
          <div className="mt-1 text-[10px] font-bold uppercase tracking-[0.22em] text-zinc-500">
            {subtitle}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2 text-[11px] font-bold uppercase tracking-[0.18em] text-zinc-400 transition-colors group-hover:text-zinc-700 dark:group-hover:text-zinc-200">
        Open
        <SlidersHorizontal size={15} />
      </div>
    </button>
  );
}

interface AudioDSPProps {
  systemStatus: string;
  onRefreshStatus: () => void;
}

export function AudioDSP({ systemStatus: _systemStatus, onRefreshStatus: _onRefreshStatus }: AudioDSPProps) {
  const [toast, setToast] = useState<{ message: string; tone?: string } | null>(null);
  const [showEqualizer, setShowEqualizer] = useState(false);
  const [showDevices, setShowDevices] = useState(false);
  const init = useAudioStore((state) => state.init);

  useEffect(() => {
    init();
  }, [init]);

  useEffect(() => {
    let toastTimer: number | undefined;
    const runtimeWindow = window as Window & { runtime?: unknown };
    if (typeof window === "undefined" || !runtimeWindow.runtime) {
      return () => {
        if (toastTimer) {
          window.clearTimeout(toastTimer);
        }
      };
    }

    const unsubscribe = EventsOn("app:toast", (payload?: { message?: string; tone?: string }) => {
      const message = payload?.message?.trim();
      if (!message) {
        return;
      }

      setToast({ message, tone: payload?.tone });
      if (toastTimer) {
        window.clearTimeout(toastTimer);
      }
      toastTimer = window.setTimeout(() => setToast(null), 3400);
    });

    return () => {
      if (toastTimer) {
        window.clearTimeout(toastTimer);
      }
      unsubscribe?.();
    };
  }, []);

  const containerVariants = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: { staggerChildren: 0.125 },
    },
  };

  const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    show: {
      opacity: 1,
      y: 0,
      transition: {
        type: "spring" as const,
        stiffness: 300,
        damping: 24,
      },
    },
  };

  return (
    <div id="App" className="h-screen w-screen flex bg-background overflow-hidden relative">
      {toast && (
        <div className="pointer-events-none absolute right-6 top-6 z-50">
          <div
            className={`min-w-[280px] max-w-[360px] rounded-2xl border px-4 py-3 shadow-xl backdrop-blur-md transition-all ${
              toast.tone === "success"
                ? "border-emerald-200 bg-emerald-50/95 text-emerald-900"
                : toast.tone === "warning"
                  ? "border-amber-200 bg-amber-50/95 text-amber-900"
                  : "border-rose-200 bg-rose-50/95 text-rose-900"
            }`}
          >
            <div className="text-[11px] font-semibold uppercase tracking-[0.18em] opacity-70">
              ViPER
            </div>
            <div className="mt-1 text-sm font-medium leading-5">{toast.message}</div>
          </div>
        </div>
      )}

      <Sidebar />

      <main className="flex-1 flex flex-col z-10 overflow-y-auto p-5 xl:p-6 custom-scrollbar">
        <StatusHeader />

        <motion.div
          variants={containerVariants}
          initial="hidden"
          animate="show"
          className="flex flex-col gap-4 pb-5 mt-4 max-w-[1520px] mx-auto w-full"
        >
          <motion.section variants={itemVariants} className="w-full">
            <MasterControls />
          </motion.section>

          <motion.section variants={itemVariants} className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
            <XBass />
            <XClarity />
            <Surround3D />
            <ReverbSidebar />
          </motion.section>

          <motion.footer variants={itemVariants} className="grid grid-cols-1 xl:grid-cols-2 gap-4 mt-1">
            <UtilityLauncher
              icon={<Activity size={18} className="text-violet-400" />}
              title="Equalizer"
              subtitle="18-Band precision control"
              accent="bg-violet-500/10"
              onClick={() => setShowEqualizer(true)}
            />
            <UtilityLauncher
              icon={<Speaker size={18} className="text-red-400" />}
              title="Audio Devices"
              subtitle="APO driver configuration"
              accent="bg-red-500/10"
              onClick={() => setShowDevices(true)}
            />
          </motion.footer>
        </motion.div>
      </main>

      {showEqualizer && (
        <OverlayShell
          size="wide"
          onClose={() => setShowEqualizer(false)}
        >
          <Equalizer onClose={() => setShowEqualizer(false)} overlay />
        </OverlayShell>
      )}

      {showDevices && (
        <OverlayShell
          size="medium"
          onClose={() => setShowDevices(false)}
        >
          <AudioDevices onClose={() => setShowDevices(false)} overlay />
        </OverlayShell>
      )}
    </div>
  );
}
