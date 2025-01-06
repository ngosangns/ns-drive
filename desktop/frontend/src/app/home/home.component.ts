import { CommonModule } from "@angular/common";
import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  OnDestroy,
  OnInit,
} from "@angular/core";
import { AppService } from "../app.service";
import { combineLatest, Subscription } from "rxjs";

@Component({
  selector: "app-home",
  standalone: true,
  imports: [CommonModule],
  templateUrl: "./home.component.html",
  styleUrl: "./home.component.css",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomeComponent implements OnInit, OnDestroy {
  private changeDetectorSub: Subscription | undefined;

  constructor(
    public readonly appService: AppService,
    private readonly cdr: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.changeDetectorSub = combineLatest([
      this.appService.data$,
      this.appService.configInfo$,
    ]).subscribe(() => this.cdr.detectChanges());
  }

  ngOnDestroy() {
    this.changeDetectorSub?.unsubscribe();
  }
}
