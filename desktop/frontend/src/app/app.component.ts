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
import { models } from "../../wailsjs/go/models.js";

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

  async pull(profile: models.Profile) {
    this.tab$.next("home");
    this.appService.pull(profile);
  }

  async push(profile: models.Profile) {
    this.tab$.next("home");
    this.appService.push(profile);
  }

  async bi(profile: models.Profile) {
    this.tab$.next("home");
    this.appService.bi(profile);
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
