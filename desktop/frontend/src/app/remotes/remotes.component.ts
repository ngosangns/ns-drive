import { CommonModule } from "@angular/common";
import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
} from "@angular/core";
import { FormsModule } from "@angular/forms";
import { BehaviorSubject, combineLatest, Subscription } from "rxjs";
import { AppService } from "../app.service";
import { MatButtonModule } from "@angular/material/button";
import { MatInputModule } from "@angular/material/input";
import { MatFormFieldModule } from "@angular/material/form-field";
import { MatSelectModule } from "@angular/material/select";
import { MatCardModule } from "@angular/material/card";
import { MatTableModule } from "@angular/material/table";
import { MatDialog, MatDialogModule } from "@angular/material/dialog";
import { MatIconModule } from "@angular/material/icon";
import {
  CreateRemoteDialogComponent,
  DeleteRemoteDialogComponent,
} from "./dialogs";

@Component({
  selector: "app-remotes",
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatSelectModule,
    MatCardModule,
    MatTableModule,
    MatDialogModule,
    MatIconModule,
    CreateRemoteDialogComponent,
    DeleteRemoteDialogComponent,
  ],
  templateUrl: "./remotes.component.html",
  styleUrl: "./remotes.component.scss",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RemotesComponent {
  Date = Date;
  remoteToDelete: any = null;
  isAddingRemote$ = new BehaviorSubject<boolean>(false);
  private subscription = new Subscription();

  saveBtnText$ = new BehaviorSubject<string>("Save ✓");

  constructor(
    public readonly appService: AppService,
    private readonly cdr: ChangeDetectorRef,
    private dialog: MatDialog
  ) {}

  ngOnInit(): void {
    this.subscription = combineLatest([
      this.appService.configInfo$,
      this.appService.remotes$,
    ]).subscribe(() => this.cdr.detectChanges());
    this.appService.getConfigInfo();
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }

  openDialog() {
    const dialogRef = this.dialog.open(CreateRemoteDialogComponent, {
      width: "500px",
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        if (result.action === "create") {
          this.handleRemoteCreate(result.data);
        } else if (result.action === "stop") {
          this.stopAddingRemote();
        }
      }
    });
  }

  openDeleteDialog(remote: any) {
    const dialogRef = this.dialog.open(DeleteRemoteDialogComponent, {
      width: "400px",
      data: { remote },
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result && result.action === "delete") {
        this.deleteRemote(result.remote.name);
      }
    });
  }

  async handleRemoteCreate(objData: Record<string, string>) {
    if (this.isAddingRemote$.value) return;

    try {
      this.isAddingRemote$.next(true);
      await this.appService.addRemote(objData);
    } catch (e) {
      alert("Error adding remote");
    } finally {
      this.isAddingRemote$.next(false);
    }
  }

  stopAddingRemote() {
    this.appService.stopAddingRemote();
  }

  async deleteRemote(name: string) {
    try {
      await this.appService.deleteRemote(name);
      await this.saveConfigInfo();
    } catch (e) {
      console.error(e);
    }
  }

  saveConfigInfo() {
    this.appService.saveConfigInfo();
    this.saveBtnText$.next("Saved ~");
    setTimeout(() => this.saveBtnText$.next("Save ✓"), 1000);
    this.cdr.detectChanges();
  }
}
