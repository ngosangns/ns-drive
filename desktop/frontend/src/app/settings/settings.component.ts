import { CommonModule } from "@angular/common";
import { ChangeDetectionStrategy, Component } from "@angular/core";
import { AppService } from "../app.service";

@Component({
  selector: "app-settings",
  standalone: true,
  imports: [CommonModule],
  templateUrl: "./settings.component.html",
  styleUrl: "./settings.component.css",
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SettingsComponent {
  constructor(public readonly appService: AppService) {}
}
