import { CommonModule } from "@angular/common";
import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  OnDestroy,
  OnInit,
} from "@angular/core";
import { RouterModule } from "@angular/router";
import { Action, AppService } from "./app.service.js";
import { Subscription } from "rxjs";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: "./app.component.html",
  styleUrl: "./app.component.css",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppComponent implements OnInit, OnDestroy {
  Action = Action;

  private changeDetectorSub: Subscription | undefined;

  constructor(
    public readonly appService: AppService,
    private readonly cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.changeDetectorSub = this.appService.currentAction$.subscribe(() =>
      this.cdr.detectChanges()
    );
  }

  ngOnDestroy() {}

  async pull() {
    this.appService.pull();
  }

  async push() {
    this.appService.push();
  }

  async bi() {
    this.appService.bi();
  }

  stopCommand() {
    this.appService.stopCommand();
  }
}
