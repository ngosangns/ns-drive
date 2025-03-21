import { Component, Inject } from "@angular/core";
import { CommonModule } from "@angular/common";
import { MatButtonModule } from "@angular/material/button";
import {
  MAT_DIALOG_DATA,
  MatDialogRef,
  MatDialogModule,
  MatDialogActions,
  MatDialogContent,
  MatDialogTitle,
} from "@angular/material/dialog";

@Component({
  selector: "app-delete-remote-dialog",
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatDialogModule,
    MatDialogContent,
    MatDialogTitle,
    MatDialogActions,
  ],
  template: `
    <h2 mat-dialog-title>Confirm Delete</h2>

    <mat-dialog-content>
      <p>
        Are you sure you want to delete <strong>{{ data.remote.name }}</strong
        >?
      </p>
    </mat-dialog-content>

    <mat-dialog-actions align="end">
      <button mat-button (click)="cancel()">Cancel</button>
      <button mat-raised-button color="warn" (click)="confirmDelete()">
        Yes, Delete It
      </button>
    </mat-dialog-actions>
  `,
})
export class DeleteRemoteDialogComponent {
  constructor(
    public dialogRef: MatDialogRef<DeleteRemoteDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { remote: any }
  ) {}

  confirmDelete() {
    this.dialogRef.close({ action: "delete", remote: this.data.remote });
  }

  cancel() {
    this.dialogRef.close();
  }
}
