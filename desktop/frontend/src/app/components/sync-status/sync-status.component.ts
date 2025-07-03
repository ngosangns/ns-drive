import { Component, Input, ChangeDetectionStrategy } from "@angular/core";
import { CommonModule } from "@angular/common";
import { SyncStatus } from "../../models/sync-status.interface";

@Component({
  selector: "app-sync-status",
  standalone: true,
  imports: [CommonModule],
  templateUrl: "./sync-status.component.html",
  styleUrl: "./sync-status.component.scss",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SyncStatusComponent {
  @Input() syncStatus: SyncStatus | null = null;
  @Input() showTitle: boolean = true;

  getStatusColor(): string {
    if (!this.syncStatus) return "primary";

    switch (this.syncStatus.status) {
      case "running":
        return "primary";
      case "completed":
        return "accent";
      case "error":
        return "warn";
      case "stopped":
        return "basic";
      default:
        return "primary";
    }
  }

  getStatusIcon(): string {
    if (!this.syncStatus) return "🔄";

    switch (this.syncStatus.status) {
      case "running":
        return "🔄";
      case "completed":
        return "✅";
      case "error":
        return "❌";
      case "stopped":
        return "⏹️";
      default:
        return "🔄";
    }
  }

  getActionIcon(): string {
    if (!this.syncStatus) return "🔄";

    switch (this.syncStatus.action) {
      case "pull":
        return "⬇️";
      case "push":
        return "⬆️";
      case "bi":
      case "bi-resync":
        return "🔄";
      default:
        return "🔄";
    }
  }

  getActionLabel(): string {
    if (!this.syncStatus) return "Sync";

    switch (this.syncStatus.action) {
      case "pull":
        return "Pull";
      case "push":
        return "Push";
      case "bi":
        return "Bi-Sync";
      case "bi-resync":
        return "Bi-Resync";
      default:
        return "Sync";
    }
  }

  formatBytes(bytes: number): string {
    if (bytes === 0) return "0 B";

    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  getProgressValue(): number {
    return this.syncStatus?.progress || 0;
  }

  hasTransferData(): boolean {
    return !!(
      this.syncStatus &&
      (this.syncStatus.files_transferred > 0 ||
        this.syncStatus.bytes_transferred > 0 ||
        this.syncStatus.total_files > 0 ||
        this.syncStatus.total_bytes > 0)
    );
  }

  hasActivityData(): boolean {
    return !!(
      this.syncStatus &&
      (this.syncStatus.checks > 0 ||
        this.syncStatus.deletes > 0 ||
        this.syncStatus.renames > 0 ||
        this.syncStatus.errors > 0)
    );
  }
}
