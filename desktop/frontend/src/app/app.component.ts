import { CommonModule } from "@angular/common";
import { ChangeDetectorRef, Component } from "@angular/core";
import { Pull, Push, StopCommand } from "../../wailsjs/go/main/App.js";
import { main } from "../../wailsjs/go/models.js";
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
  currentPid: number = 0;
  data: string[] = [];
  isPulling: boolean = false;
  isPushing: boolean = false;

  constructor(private cdr: ChangeDetectorRef) {}

  async ngOnInit() {
    EventsOn("tofe", (data: any) => {
      data = <CommandDTO>JSON.parse(data);

      switch (data.command) {
        case main.Command.command_started:
          this;
          this.addToData("Command started...");
          break;
        case main.Command.command_stoped:
          this.onCommandStopped();
          break;
        case main.Command.command_output:
          this.addToData(data.error);
          break;
        case main.Command.error:
          this.addToData(data);
          break;
      }
    });
  }

  addToData(str: string) {
    this.data = [str];
    this.cdr.detectChanges();
  }

  onCommandStopped() {
    this.isPulling = false;
    this.isPushing = false;
    this.currentPid = 0;
    this.addToData("Done!");
    this.cdr.detectChanges();
  }

  ngOnDestroy() {}

  async pullClick() {
    if (this.isPulling) return;

    this.data = ["Pulling..."];
    await Pull();
    this.isPulling = true;
    this.cdr.detectChanges();
  }

  async pushClick() {
    if (this.isPushing) return;

    this.data = ["Pushing..."];
    await Push();
    this.isPushing = true;
    this.cdr.detectChanges();
  }

  stopCommand() {
    if (!this.isPulling && !this.isPushing) return;
    StopCommand(this.currentPid);
  }
}
