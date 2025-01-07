import { CommonModule } from "@angular/common";
import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  OnDestroy,
  OnInit,
  ViewEncapsulation,
} from "@angular/core";
import { Action, AppService } from "./app.service.js";
import { BehaviorSubject, combineLatest, Subscription } from "rxjs";
import { SettingsComponent } from "./settings/settings.component";
import { HomeComponent } from "./home/home.component.js";

type Tab = "home" | "settings";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [CommonModule, SettingsComponent, HomeComponent, SettingsComponent],
  templateUrl: "./app.component.html",
  styleUrl: "./app.component.scss",
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None
})
export class AppComponent implements OnInit, OnDestroy {
  Action = Action;

  readonly tab$ = new BehaviorSubject<Tab>("home");

  private changeDetectorSub: Subscription | undefined;

  constructor(
    public readonly appService: AppService,
    private readonly cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.changeDetectorSub = combineLatest([
      this.appService.currentAction$,
      this.tab$,
    ]).subscribe(() => this.cdr.detectChanges());
  }

  ngOnDestroy() {}

  async pull() {
    this.tab$.next("home");
    this.appService.pull();
  }

  async push() {
    this.tab$.next("home");
    this.appService.push();
  }

  async bi() {
    this.tab$.next("home");
    this.appService.bi();
  }

  stopCommand() {
    this.appService.stopCommand();
  }

  openHome() {
    this.tab$.next("home");
  }

  openSettings() {
    this.tab$.next("settings");
  }
}
