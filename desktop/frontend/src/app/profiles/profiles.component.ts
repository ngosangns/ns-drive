import { CommonModule } from "@angular/common";
import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  OnDestroy,
  OnInit,
} from "@angular/core";
import { AppService } from "../app.service";
import { BehaviorSubject, Subscription } from "rxjs";
import { FormsModule } from "@angular/forms";
import { MatButtonModule } from "@angular/material/button";
import { MatInputModule } from "@angular/material/input";
import { MatFormFieldModule } from "@angular/material/form-field";
import { MatExpansionModule } from "@angular/material/expansion";
import { MatCardModule } from "@angular/material/card";
import { MatDividerModule } from "@angular/material/divider";
import { MatIconModule } from "@angular/material/icon";

@Component({
  selector: "app-profiles",
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatExpansionModule,
    MatCardModule,
    MatDividerModule,
    MatIconModule,
  ],
  templateUrl: "./profiles.component.html",
  styleUrl: "./profiles.component.scss",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfilesComponent implements OnInit, OnDestroy {
  Date = Date;
  private changeDetectorSub: Subscription | undefined;

  saveBtnText$ = new BehaviorSubject<string>("Save ✓");

  constructor(
    public readonly appService: AppService,
    private readonly cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.changeDetectorSub = this.appService.configInfo$.subscribe(() =>
      this.cdr.detectChanges()
    );
    this.appService.getConfigInfo();
  }

  ngOnDestroy(): void {}

  addProfile() {
    this.appService.addProfile();
  }

  removeProfile(idx: number) {
    this.appService.removeProfile(idx);
  }

  saveConfigInfo() {
    this.appService.saveConfigInfo();
    this.saveBtnText$.next("Saved ~");
    setTimeout(() => this.saveBtnText$.next("Save ✓"), 1000);
    this.cdr.detectChanges();
  }

  addIncludePath(profileIndex: number) {
    this.appService.addIncludePath(profileIndex);
    this.cdr.detectChanges();
  }

  removeIncludePath(profileIndex: number, idx: number) {
    this.appService.removeIncludePath(profileIndex, idx);
    this.cdr.detectChanges();
  }

  addExcludePath(profileIndex: number) {
    this.appService.addExcludePath(profileIndex);
    this.cdr.detectChanges();
  }

  removeExcludePath(profileIndex: number, idx: number) {
    this.appService.removeExcludePath(profileIndex, idx);
    this.cdr.detectChanges();
  }

  trackByFn(index: number, item: any): number {
    return index;
  }
}
