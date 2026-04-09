import { useEffect, useState } from "react";
import { motion } from "framer-motion";

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

interface AudioDSPProps {
  systemStatus: string;
  onRefreshStatus: () => void;
}

export function AudioDSP({ systemStatus: _systemStatus, onRefreshStatus: _onRefreshStatus }: AudioDSPProps) {
  const [toast, setToast] = useState<{ message: string; tone?: string } | null>(null);
  const init = useAudioStore((state) => state.init);

  useEffect(() => {
    init();
  }, [init]);

  useEffect(() => {
    let toastTimer: number | undefined;
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

      <main className="flex-1 flex flex-col z-10 overflow-y-auto p-8 custom-scrollbar">
        <StatusHeader />

        <motion.div
          variants={containerVariants}
          initial="hidden"
          animate="show"
          className="flex flex-col gap-5 pb-6 mt-8 max-w-[1600px] mx-auto w-full"
        >
          <motion.section variants={itemVariants} className="w-full">
            <MasterControls />
          </motion.section>

          <motion.section variants={itemVariants} className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-5">
            <XBass />
            <XClarity />
            <Surround3D />
            <ReverbSidebar />
          </motion.section>

          <motion.footer variants={itemVariants} className="grid grid-cols-2 gap-5 mt-2">
            <Equalizer />
            <AudioDevices />
          </motion.footer>
        </motion.div>
      </main>
    </div>
  );
}
