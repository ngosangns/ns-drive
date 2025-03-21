"use client";

import React, { useEffect, useMemo } from "react";
import { Layout } from "@/components/layout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Action, useAppContext } from "@/lib/app-context";

export default function Home() {
  const appContext = useAppContext();
  const {
    currentAction,
    configInfo,
    data,
    pull,
    push,
    bi,
    stopCommand,
    setConfigInfo,
  } = appContext;

  const isCurrentProfileValid = useMemo(() => {
    if (configInfo.selected_profile_index === null) return false;
    const profileIndex = configInfo.selected_profile_index;
    const profile = configInfo.profiles[profileIndex];
    return profile && profile.name && profile.from && profile.to;
  }, [configInfo]);

  const handlePull = () => {
    if (isCurrentProfileValid && configInfo.selected_profile_index !== null) {
      const profile = configInfo.profiles[configInfo.selected_profile_index];
      pull(profile);
    }
  };

  const handlePush = () => {
    if (isCurrentProfileValid && configInfo.selected_profile_index !== null) {
      const profile = configInfo.profiles[configInfo.selected_profile_index];
      push(profile);
    }
  };

  const handleBi = () => {
    if (isCurrentProfileValid && configInfo.selected_profile_index !== null) {
      const profile = configInfo.profiles[configInfo.selected_profile_index];
      bi(profile);
    }
  };

  const handleBiResync = () => {
    if (isCurrentProfileValid && configInfo.selected_profile_index !== null) {
      const profile = configInfo.profiles[configInfo.selected_profile_index];
      bi(profile, true);
    }
  };

  const handleProfileChange = (value: string) => {
    const index = value === "" ? null : parseInt(value, 10);
    setConfigInfo({
      ...configInfo,
      selected_profile_index: index,
    });
  };

  useEffect(() => {
    // Add any initialization here
  }, []);

  return (
    <Layout>
      <div className="h-screen p-6 pl-3 overflow-x-hidden overflow-y-auto">
        <Card className="mb-4">
          <CardHeader>
            <CardTitle>Actions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-3 items-center">
              <Button
                variant={currentAction === Action.Pull ? "accent" : "default"}
                onClick={
                  currentAction !== Action.Pull ? handlePull : stopCommand
                }
                disabled={!isCurrentProfileValid}
              >
                {currentAction === Action.Pull ? "Stop" : "Pull"}
              </Button>

              <Button
                variant={currentAction === Action.Push ? "accent" : "default"}
                onClick={
                  currentAction !== Action.Push ? handlePush : stopCommand
                }
                disabled={!isCurrentProfileValid}
              >
                {currentAction === Action.Push ? "Stop" : "Push"}
              </Button>

              <Button
                variant={currentAction === Action.Bi ? "accent" : "default"}
                onClick={currentAction !== Action.Bi ? handleBi : stopCommand}
                disabled={!isCurrentProfileValid}
              >
                {currentAction === Action.Bi ? "Stop" : "Sync"}
              </Button>

              <Button
                variant={
                  currentAction === Action.BiResync ? "accent" : "default"
                }
                onClick={
                  currentAction !== Action.BiResync
                    ? handleBiResync
                    : stopCommand
                }
                disabled={!isCurrentProfileValid}
              >
                {currentAction === Action.BiResync ? "Stop" : "Resync"}
              </Button>

              <Select
                value={
                  configInfo.selected_profile_index === null
                    ? ""
                    : configInfo.selected_profile_index.toString()
                }
                onValueChange={handleProfileChange}
              >
                <SelectTrigger className="w-[200px]">
                  <SelectValue placeholder="Select a profile" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">Profile is not selected</SelectItem>
                  {configInfo.profiles.map((profile, index) => (
                    <SelectItem key={index} value={index.toString()}>
                      {profile.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        <Card className="mb-4">
          <CardContent>
            <pre className="text-white">
              Working directory: {configInfo.working_dir}
            </pre>
          </CardContent>
        </Card>

        <Card className="flex-grow">
          <CardContent>
            <pre className="text-white h-full overflow-auto">
              {data.join("\n")}
            </pre>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
