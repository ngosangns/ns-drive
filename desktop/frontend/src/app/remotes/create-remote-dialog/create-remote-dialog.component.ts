import { Component, Inject } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { MatButtonModule } from "@angular/material/button";
import { MatInputModule } from "@angular/material/input";
import { MatFormFieldModule } from "@angular/material/form-field";
import { MatSelectModule } from "@angular/material/select";
import {
  MAT_DIALOG_DATA,
  MatDialogRef,
  MatDialogModule,
  MatDialogActions,
  MatDialogContent,
  MatDialogTitle,
} from "@angular/material/dialog";
import { BehaviorSubject } from "rxjs";

@Component({
  selector: "app-create-remote-dialog",
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatSelectModule,
    MatDialogModule,
    MatDialogContent,
    MatDialogTitle,
    MatDialogActions,
  ],
  template: `
    <h2 mat-dialog-title>Add New Remote</h2>
    <form (submit)="submitForm($event)" class="p-4">
      <mat-dialog-content>
        <div class="grid grid-cols-2 gap-4">
          <mat-form-field appearance="outline">
            <mat-label>Name</mat-label>
            <input matInput name="name" placeholder="google-drive" />
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Type</mat-label>
            <mat-select name="type">
              <mat-option value="drive">Google Drive</mat-option>
              <mat-option value="dropbox">Dropbox</mat-option>
              <mat-option value="onedrive">Onedrive</mat-option>
              <mat-option value="yandex">Yandex Disk</mat-option>
              <mat-option value="gphotos">Google Photos</mat-option>
            </mat-select>
          </mat-form-field>
        </div>
      </mat-dialog-content>

      <mat-dialog-actions align="end">
        <button mat-button type="button" (click)="cancel()" *ngIf="!isAdding">
          Cancel
        </button>
        <button
          mat-raised-button
          color="primary"
          type="submit"
          *ngIf="!isAdding"
        >
          Create
        </button>
        <button
          mat-raised-button
          color="warn"
          type="button"
          *ngIf="isAdding"
          (click)="stop()"
        >
          Stop
        </button>
      </mat-dialog-actions>
    </form>
  `,
})
export class CreateRemoteDialogComponent {
  isAdding = false;

  constructor(
    public dialogRef: MatDialogRef<CreateRemoteDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {}

  submitForm(e: SubmitEvent) {
    e.preventDefault();

    const data = new FormData(e.target as HTMLFormElement);
    const objData: Record<string, string> = [...(data as any).entries()].reduce(
      (acc, [key, value]) => {
        acc[key] = value;
        return acc;
      },
      {}
    );

    this.isAdding = true;
    this.dialogRef.close({ action: "create", data: objData });
  }

  stop() {
    this.dialogRef.close({ action: "stop" });
  }

  cancel() {
    this.dialogRef.close();
  }
}
