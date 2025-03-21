"use client";

import React, { useState } from "react";
import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { useAppContext } from "@/lib/app-context";

export default function ProfilesPage() {
  const {
    configInfo,
    addProfile,
    removeProfile,
    saveConfigInfo,
    addIncludePath,
    removeIncludePath,
    addExcludePath,
    removeExcludePath,
    setConfigInfo,
  } = useAppContext();

  const [saveBtnText, setSaveBtnText] = useState<string>("Save");

  const handleSaveConfigInfo = async () => {
    setSaveBtnText("Saving...");
    await saveConfigInfo();
    setSaveBtnText("Saved!");

    setTimeout(() => {
      setSaveBtnText("Save");
    }, 2000);
  };

  const updateProfileField = (
    profileIndex: number,
    field: string,
    value: any
  ) => {
    const updatedProfiles = [...configInfo.profiles];
    updatedProfiles[profileIndex] = {
      ...updatedProfiles[profileIndex],
      [field]: value,
    };
    setConfigInfo({
      ...configInfo,
      profiles: updatedProfiles,
    });
  };

  return (
    <Layout>
      <div className="h-screen p-6 pl-3 overflow-x-hidden overflow-y-auto">
        <Card>
          <CardContent className="pt-6">
            <div className="flex gap-3 mb-4">
              <Button onClick={addProfile}>Add Profile</Button>
              <div className="grow" />
              <Button variant="accent" onClick={handleSaveConfigInfo}>
                {saveBtnText}
              </Button>
            </div>

            <Accordion type="multiple" className="w-full">
              {configInfo.profiles.map((profile, idx) => (
                <AccordionItem key={idx} value={`profile-${idx}`}>
                  <AccordionTrigger>
                    {profile.name || "No title"}
                  </AccordionTrigger>
                  <AccordionContent>
                    <div className="p-3">
                      <div className="mb-4">
                        <label className="text-sm font-medium mb-1 block">
                          Name
                        </label>
                        <Input
                          value={profile.name}
                          onChange={(e) =>
                            updateProfileField(idx, "name", e.target.value)
                          }
                          placeholder="No title"
                          className="w-full"
                        />
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <label className="text-sm font-medium mb-1 block">
                            From path
                          </label>
                          <Input
                            value={profile.from}
                            onChange={(e) =>
                              updateProfileField(idx, "from", e.target.value)
                            }
                            placeholder="google-drive:/drive"
                          />
                        </div>
                        <div>
                          <label className="text-sm font-medium mb-1 block">
                            To path
                          </label>
                          <Input
                            value={profile.to}
                            onChange={(e) =>
                              updateProfileField(idx, "to", e.target.value)
                            }
                            placeholder="./drive"
                          />
                        </div>
                        <div>
                          <label className="text-sm font-medium mb-1 block">
                            Backup path
                          </label>
                          <Input
                            value={profile.backup_path}
                            onChange={(e) =>
                              updateProfileField(
                                idx,
                                "backup_path",
                                e.target.value
                              )
                            }
                            placeholder=".backup"
                          />
                        </div>
                        <div>
                          <label className="text-sm font-medium mb-1 block">
                            Cache path
                          </label>
                          <Input
                            value={profile.cache_path}
                            onChange={(e) =>
                              updateProfileField(
                                idx,
                                "cache_path",
                                e.target.value
                              )
                            }
                            placeholder=".cache"
                          />
                        </div>
                        <div>
                          <label className="text-sm font-medium mb-1 block">
                            Parallel
                          </label>
                          <Input
                            type="number"
                            value={profile.parallel.toString()}
                            onChange={(e) =>
                              updateProfileField(
                                idx,
                                "parallel",
                                parseInt(e.target.value) || 1
                              )
                            }
                            placeholder="16"
                            min="1"
                          />
                        </div>
                        <div>
                          <label className="text-sm font-medium mb-1 block">
                            Bandwidth (MB/s)
                          </label>
                          <Input
                            type="number"
                            value={profile.bandwidth.toString()}
                            onChange={(e) =>
                              updateProfileField(
                                idx,
                                "bandwidth",
                                parseInt(e.target.value) || 1
                              )
                            }
                            placeholder="5"
                            min="1"
                          />
                        </div>
                      </div>

                      <Card className="mt-4">
                        <CardContent className="pt-4">
                          <h3 className="text-lg font-medium mb-2">
                            Include paths
                          </h3>
                          {profile.included_paths.map((path, pathIdx) => (
                            <div
                              key={pathIdx}
                              className="flex items-center gap-2 mb-2"
                            >
                              <Input
                                value={path}
                                onChange={(e) => {
                                  const newPaths = [...profile.included_paths];
                                  newPaths[pathIdx] = e.target.value;
                                  updateProfileField(
                                    idx,
                                    "included_paths",
                                    newPaths
                                  );
                                }}
                                placeholder="/included/**"
                                className="flex-1"
                              />
                              <Button
                                variant="destructive"
                                size="icon"
                                onClick={() => removeIncludePath(idx, pathIdx)}
                              >
                                X
                              </Button>
                            </div>
                          ))}
                          <Button
                            variant="outline"
                            onClick={() => addIncludePath(idx)}
                            className="mt-2"
                          >
                            Add path
                          </Button>
                        </CardContent>
                      </Card>

                      <Card className="mt-4">
                        <CardContent className="pt-4">
                          <h3 className="text-lg font-medium mb-2">
                            Exclude paths
                          </h3>
                          {profile.excluded_paths.map((path, pathIdx) => (
                            <div
                              key={pathIdx}
                              className="flex items-center gap-2 mb-2"
                            >
                              <Input
                                value={path}
                                onChange={(e) => {
                                  const newPaths = [...profile.excluded_paths];
                                  newPaths[pathIdx] = e.target.value;
                                  updateProfileField(
                                    idx,
                                    "excluded_paths",
                                    newPaths
                                  );
                                }}
                                placeholder="/excluded/**"
                                className="flex-1"
                              />
                              <Button
                                variant="destructive"
                                size="icon"
                                onClick={() => removeExcludePath(idx, pathIdx)}
                              >
                                X
                              </Button>
                            </div>
                          ))}
                          <Button
                            variant="outline"
                            onClick={() => addExcludePath(idx)}
                            className="mt-2"
                          >
                            Add path
                          </Button>
                        </CardContent>
                      </Card>

                      <div className="mt-4">
                        <Button
                          variant="destructive"
                          onClick={() => removeProfile(idx)}
                        >
                          Delete Profile
                        </Button>
                      </div>
                    </div>
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
