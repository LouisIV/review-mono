import { useState } from "react";
import SettingsPage from "./pages/SettingsPage";
import DaemonPage from "./pages/DaemonPage";

type Tab = "daemons" | "settings";

export default function App() {
  const [tab, setTab] = useState<Tab>("daemons");

  return (
    <div className="flex flex-col h-screen">
      <nav className="flex border-b bg-white px-4 gap-1">
        <button onClick={() => setTab("daemons")}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${tab === "daemons" ? "border-blue-600 text-blue-600" : "border-transparent text-gray-500 hover:text-gray-700"}`}>
          Daemons
        </button>
        <button onClick={() => setTab("settings")}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${tab === "settings" ? "border-blue-600 text-blue-600" : "border-transparent text-gray-500 hover:text-gray-700"}`}>
          Settings
        </button>
      </nav>
      <main className="flex-1 overflow-hidden">
        {tab === "daemons" && <DaemonPage />}
        {tab === "settings" && <SettingsPage />}
      </main>
    </div>
  );
}
