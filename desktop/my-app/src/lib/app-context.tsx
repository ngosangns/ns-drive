"use client";

import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";

export enum Action {
  Pull = "pull",
  Push = "push",
  Bi = "bi",
  BiResync = "bi-resync",
}

export interface Remote {
  name: string;
  type: string;
}

export interface Profile {
  name: string;
  from: string;
  to: string;
  backup_path: string;
  cache_path: string;
  parallel: number;
  bandwidth: number;
  included_paths: string[];
  excluded_paths: string[];
}

export interface ConfigInfo {
  working_dir: string;
  profiles: Profile[];
  selected_profile_index: number | null;
}

interface AppContextType {
  currentAction: Action | null;
  configInfo: ConfigInfo;
  remotes: Remote[];
  data: string[];
  tab: string;
  setTab: (tab: string) => void;
  setCurrentAction: (action: Action | null) => void;
  setConfigInfo: (configInfo: ConfigInfo) => void;
  setRemotes: (remotes: Remote[]) => void;
  setData: (data: string[]) => void;
  replaceData: (str: string) => void;
  pull: (profile: Profile) => Promise<void>;
  push: (profile: Profile) => Promise<void>;
  bi: (profile: Profile, resync?: boolean) => Promise<void>;
  stopCommand: () => void;
  getConfigInfo: () => Promise<void>;
  getRemotes: () => Promise<void>;
  addRemote: (objData: Record<string, string>) => Promise<void>;
  stopAddingRemote: () => Promise<void>;
  deleteRemote: (name: string) => Promise<void>;
  addProfile: () => void;
  removeProfile: (index: number) => void;
  saveConfigInfo: () => Promise<void>;
  addIncludePath: (profileIndex: number) => void;
  removeIncludePath: (profileIndex: number, index: number) => void;
  addExcludePath: (profileIndex: number) => void;
  removeExcludePath: (profileIndex: number, index: number) => void;
}

const defaultConfigInfo: ConfigInfo = {
  working_dir: "",
  profiles: [],
  selected_profile_index: null,
};

const AppContext = createContext<AppContextType | undefined>(undefined);

export function AppProvider({ children }: { children: ReactNode }) {
  const [currentAction, setCurrentAction] = useState<Action | null>(null);
  const [configInfo, setConfigInfo] = useState<ConfigInfo>(defaultConfigInfo);
  const [remotes, setRemotes] = useState<Remote[]>([]);
  const [data, setData] = useState<string[]>([]);
  const [tab, setTab] = useState<string>("home");

  // Mock implementations for now - in a real app these would interact with backend APIs
  const replaceData = (str: string) => {
    setData((prev) => [...prev, str]);
  };

  const pull = async (profile: Profile) => {
    setCurrentAction(Action.Pull);
    replaceData(`Running pull operation with profile: ${profile.name}`);
    // Implementation would go here
  };

  const push = async (profile: Profile) => {
    setCurrentAction(Action.Push);
    replaceData(`Running push operation with profile: ${profile.name}`);
    // Implementation would go here
  };

  const bi = async (profile: Profile, resync = false) => {
    setCurrentAction(resync ? Action.BiResync : Action.Bi);
    replaceData(
      `Running ${resync ? "resync" : "sync"} operation with profile: ${
        profile.name
      }`
    );
    // Implementation would go here
  };

  const stopCommand = () => {
    setCurrentAction(null);
    replaceData("Operation stopped");
    // Implementation would go here
  };

  const getConfigInfo = async () => {
    // Mock implementation - would fetch from API
    replaceData("Fetching config info");
    // Implementation would go here
  };

  const getRemotes = async () => {
    // Mock implementation - would fetch from API
    replaceData("Fetching remotes");
    // Implementation would go here
  };

  const addRemote = async (objData: Record<string, string>) => {
    // Mock implementation
    const newRemote: Remote = {
      name: objData.name || "New Remote",
      type: objData.type || "unknown",
    };
    setRemotes((prev) => [...prev, newRemote]);
    replaceData(`Added new remote: ${newRemote.name}`);
  };

  const stopAddingRemote = async () => {
    replaceData("Stopped adding remote");
    // Implementation would go here
  };

  const deleteRemote = async (name: string) => {
    setRemotes((prev) => prev.filter((remote) => remote.name !== name));
    replaceData(`Deleted remote: ${name}`);
  };

  const addProfile = () => {
    const newProfile: Profile = {
      name: "",
      from: "",
      to: "",
      backup_path: ".backup",
      cache_path: ".cache",
      parallel: 16,
      bandwidth: 5,
      included_paths: [],
      excluded_paths: [],
    };
    setConfigInfo((prev) => ({
      ...prev,
      profiles: [...prev.profiles, newProfile],
    }));
  };

  const removeProfile = (index: number) => {
    setConfigInfo((prev) => ({
      ...prev,
      profiles: prev.profiles.filter((_, i) => i !== index),
    }));
  };

  const saveConfigInfo = async () => {
    replaceData("Saving config info");
    // Implementation would go here
  };

  const addIncludePath = (profileIndex: number) => {
    setConfigInfo((prev) => {
      const profiles = [...prev.profiles];
      if (profiles[profileIndex]) {
        profiles[profileIndex] = {
          ...profiles[profileIndex],
          included_paths: [...profiles[profileIndex].included_paths, ""],
        };
      }
      return { ...prev, profiles };
    });
  };

  const removeIncludePath = (profileIndex: number, index: number) => {
    setConfigInfo((prev) => {
      const profiles = [...prev.profiles];
      if (profiles[profileIndex]) {
        profiles[profileIndex] = {
          ...profiles[profileIndex],
          included_paths: profiles[profileIndex].included_paths.filter(
            (_, i) => i !== index
          ),
        };
      }
      return { ...prev, profiles };
    });
  };

  const addExcludePath = (profileIndex: number) => {
    setConfigInfo((prev) => {
      const profiles = [...prev.profiles];
      if (profiles[profileIndex]) {
        profiles[profileIndex] = {
          ...profiles[profileIndex],
          excluded_paths: [...profiles[profileIndex].excluded_paths, ""],
        };
      }
      return { ...prev, profiles };
    });
  };

  const removeExcludePath = (profileIndex: number, index: number) => {
    setConfigInfo((prev) => {
      const profiles = [...prev.profiles];
      if (profiles[profileIndex]) {
        profiles[profileIndex] = {
          ...profiles[profileIndex],
          excluded_paths: profiles[profileIndex].excluded_paths.filter(
            (_, i) => i !== index
          ),
        };
      }
      return { ...prev, profiles };
    });
  };

  // Initialize on component mount
  useEffect(() => {
    // Initialize with some mock data
    setConfigInfo({
      working_dir: "/Users/username/Documents",
      profiles: [
        {
          name: "Sample Profile",
          from: "google-drive:/drive",
          to: "./drive",
          backup_path: ".backup",
          cache_path: ".cache",
          parallel: 16,
          bandwidth: 5,
          included_paths: ["/included/**"],
          excluded_paths: ["/excluded/**"],
        },
      ],
      selected_profile_index: 0,
    });

    setRemotes([
      { name: "Google Drive", type: "google-drive" },
      { name: "Dropbox", type: "dropbox" },
    ]);
  }, []);

  return (
    <AppContext.Provider
      value={{
        currentAction,
        configInfo,
        remotes,
        data,
        tab,
        setTab,
        setCurrentAction,
        setConfigInfo,
        setRemotes,
        setData,
        replaceData,
        pull,
        push,
        bi,
        stopCommand,
        getConfigInfo,
        getRemotes,
        addRemote,
        stopAddingRemote,
        deleteRemote,
        addProfile,
        removeProfile,
        saveConfigInfo,
        addIncludePath,
        removeIncludePath,
        addExcludePath,
        removeExcludePath,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}

export function useAppContext() {
  const context = useContext(AppContext);
  if (context === undefined) {
    throw new Error("useAppContext must be used within an AppProvider");
  }
  return context;
}
