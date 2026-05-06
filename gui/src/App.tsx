import { Server, Settings } from "lucide-react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import DaemonPage from "@/pages/DaemonPage";
import SettingsPage from "@/pages/SettingsPage";

export default function App() {
  return (
    <div className="flex flex-col h-screen bg-background">
      <Tabs defaultValue="daemons" className="flex flex-col h-full">
        <header className="border-b px-4 py-2">
          <TabsList>
            <TabsTrigger value="daemons">
              <Server className="h-4 w-4" />
              Daemons
            </TabsTrigger>
            <TabsTrigger value="settings">
              <Settings className="h-4 w-4" />
              Settings
            </TabsTrigger>
          </TabsList>
        </header>
        <TabsContent value="daemons" className="flex-1 overflow-hidden m-0">
          <DaemonPage />
        </TabsContent>
        <TabsContent value="settings" className="flex-1 overflow-hidden m-0">
          <SettingsPage />
        </TabsContent>
      </Tabs>
    </div>
  );
}
