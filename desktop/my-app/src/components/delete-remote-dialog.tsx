import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Remote } from "@/lib/app-context";

interface DeleteRemoteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  remote: Remote | null;
  onConfirm: (name: string) => void;
  onCancel: () => void;
}

export function DeleteRemoteDialog({
  open,
  onOpenChange,
  remote,
  onConfirm,
  onCancel,
}: DeleteRemoteDialogProps) {
  const handleConfirm = () => {
    if (remote) {
      onConfirm(remote.name);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Remote</DialogTitle>
        </DialogHeader>

        <div className="py-4">
          <p>Are you sure you want to delete the remote "{remote?.name}"?</p>
          <p className="text-muted-foreground mt-2">
            This action cannot be undone.
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={handleConfirm}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
