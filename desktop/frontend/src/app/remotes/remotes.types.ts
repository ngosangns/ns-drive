/**
 * Type definitions for the remotes component
 */

export type RemoteType =
  | "drive"
  | "dropbox"
  | "onedrive"
  | "yandex"
  | "gphotos"
  | "iclouddrive";

export interface RemoteFormData {
  name: string;
  type: RemoteType;
}

export interface RemoteTypeOption {
  value: RemoteType;
  label: string;
  icon: string;
}

export interface ConfirmDeleteDialogData {
  remoteName: string;
}

export interface RemoteInfo {
  name: string;
  type: string;
  source?: string;
  description?: string;
}

// Constants for remote type options
export const REMOTE_TYPE_OPTIONS: RemoteTypeOption[] = [
  { value: "drive", label: "Google Drive", icon: "cloud" },
  { value: "dropbox", label: "Dropbox", icon: "cloud_queue" },
  { value: "onedrive", label: "OneDrive", icon: "cloud_circle" },
  { value: "yandex", label: "Yandex Disk", icon: "cloud_sync" },
  { value: "gphotos", label: "Google Photos", icon: "photo_library" },
  { value: "iclouddrive", label: "iCloud Drive", icon: "cloud_upload" },
];

// Type guards
export function isValidRemoteType(type: string): type is RemoteType {
  return [
    "drive",
    "dropbox",
    "onedrive",
    "yandex",
    "gphotos",
    "iclouddrive",
  ].includes(type);
}

export function isRemoteFormData(data: unknown): data is RemoteFormData {
  return (
    data !== null &&
    typeof data === "object" &&
    data !== undefined &&
    "name" in data &&
    "type" in data &&
    typeof (data as RemoteFormData).name === "string" &&
    typeof (data as RemoteFormData).type === "string" &&
    isValidRemoteType((data as RemoteFormData).type)
  );
}
