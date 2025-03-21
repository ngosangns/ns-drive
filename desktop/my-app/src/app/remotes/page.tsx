"use client";

import React, { useState } from "react";
import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useAppContext, Remote } from "@/lib/app-context";
import { CreateRemoteDialog } from "@/components/create-remote-dialog";
import { DeleteRemoteDialog } from "@/components/delete-remote-dialog";

export default function RemotesPage() {
  const { remotes, addRemote, deleteRemote, stopAddingRemote } =
    useAppContext();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedRemote, setSelectedRemote] = useState<Remote | null>(null);

  const handleOpenCreateDialog = () => {
    setCreateDialogOpen(true);
  };

  const handleCloseCreateDialog = () => {
    setCreateDialogOpen(false);
  };

  const handleOpenDeleteDialog = (remote: Remote) => {
    setSelectedRemote(remote);
    setDeleteDialogOpen(true);
  };

  const handleCloseDeleteDialog = () => {
    setSelectedRemote(null);
    setDeleteDialogOpen(false);
  };

  const handleSubmitRemote = async (data: Record<string, string>) => {
    await addRemote(data);
    setCreateDialogOpen(false);
  };

  const handleCancelCreateRemote = () => {
    stopAddingRemote();
    setCreateDialogOpen(false);
  };

  const handleConfirmDelete = async (name: string) => {
    await deleteRemote(name);
    setDeleteDialogOpen(false);
    setSelectedRemote(null);
  };

  return (
    <Layout>
      <div className="h-screen p-6 pl-3 overflow-x-hidden overflow-y-auto">
        <Card>
          <CardContent className="pt-6">
            <div className="flex justify-between mb-4">
              <Button onClick={handleOpenCreateDialog}>Add Remote</Button>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">Name</th>
                    <th className="text-left p-2">Type</th>
                    <th className="w-24 text-right p-2">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {remotes.map((remote, index) => (
                    <tr key={index} className="border-b">
                      <td className="p-2">{remote.name}</td>
                      <td className="p-2">{remote.type}</td>
                      <td className="p-2 text-right">
                        <Button
                          variant="destructive"
                          size="icon"
                          onClick={() => handleOpenDeleteDialog(remote)}
                        >
                          X
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <CreateRemoteDialog
              open={createDialogOpen}
              onOpenChange={setCreateDialogOpen}
              onSubmit={handleSubmitRemote}
              onCancel={handleCancelCreateRemote}
            />

            <DeleteRemoteDialog
              open={deleteDialogOpen}
              onOpenChange={setDeleteDialogOpen}
              remote={selectedRemote}
              onConfirm={handleConfirmDelete}
              onCancel={handleCloseDeleteDialog}
            />
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
