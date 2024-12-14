import { CommonModule } from "@angular/common";
import { ChangeDetectorRef, Component } from "@angular/core";
import { Pull, Push, StopCommand } from "../../wailsjs/go/backend/App.js";
import { dto } from "../../wailsjs/go/models.js";
import { EventsOn } from "../../wailsjs/runtime";

interface CommandDTO {
  command: string;
  pid: number | undefined;
  task: string | undefined;
  error: string | undefined;
}

@Component({
  selector: "app-root",
  standalone: true,
  imports: [CommonModule],
  templateUrl: "./app.component.html",
  styleUrl: "./app.component.css",
})
export class AppComponent {
  currentId: number = 0;
  data: string[] = [];
  isPulling: boolean = false;
  isPushing: boolean = false;

  constructor(private cdr: ChangeDetectorRef) {}

  async ngOnInit() {
    EventsOn("tofe", (data: any) => {
      data = <CommandDTO>JSON.parse(data);

      switch (data.command) {
        case dto.Command.command_started:
          this;
          this.replaceData("Command started...");
          break;
        case dto.Command.command_stoped:
          this.onCommandStopped();
          break;
        case dto.Command.command_output:
          this.replaceData(data.error);
          break;
        case dto.Command.error:
          this.addData(data);
          break;
      }
    });
  }

  replaceData(str: string) {
    this.data = [str];
    this.cdr.detectChanges();
  }

  addData(str: string) {
    this.data.push(str);
    this.cdr.detectChanges();
  }

  onCommandStopped() {
    this.isPulling = false;
    this.isPushing = false;
    this.currentId = 0;
    this.cdr.detectChanges();
  }

  ngOnDestroy() {}

  async pullClick() {
    if (this.isPulling) return;

    this.replaceData("Pulling...");
    this.currentId = await Pull();
    if (this.currentId) {
      this.isPulling = true;
      this.cdr.detectChanges();
    }
  }

  async pushClick() {
    if (this.isPushing) return;

    this.replaceData("Pushing...");
    this.currentId = await Push();
    if (this.currentId) {
      this.isPushing = true;
      this.cdr.detectChanges();
    }
  }

  stopCommand() {
    if (!this.isPulling && !this.isPushing) return;
    StopCommand(this.currentId);
  }
}
