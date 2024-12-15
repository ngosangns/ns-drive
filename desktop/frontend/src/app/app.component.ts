import { CommonModule } from "@angular/common";
import { ChangeDetectorRef, Component } from "@angular/core";
import { Sync, StopCommand } from "../../wailsjs/go/backend/App.js";
import { dto } from "../../wailsjs/go/models.js";
import { EventsOn } from "../../wailsjs/runtime";

interface CommandDTO {
  command: string;
  pid: number | undefined;
  task: string | undefined;
  error: string | undefined;
}

enum Action {
  Pull = "pull",
  Push = "push",
  Bi = "bi",
}

@Component({
  selector: "app-root",
  standalone: true,
  imports: [CommonModule],
  templateUrl: "./app.component.html",
  styleUrl: "./app.component.css",
})
export class AppComponent {
  Action = Action;
  currentId: number = 0;
  currentAction: Action | undefined;

  data: string[] = [];

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
    this.currentAction = undefined;
    this.currentId = 0;
    this.cdr.detectChanges();
  }

  ngOnDestroy() {}

  async pullClick() {
    if (this.currentAction === Action.Pull) return;

    this.replaceData("Pulling...");
    this.currentId = await Sync("pull");
    if (this.currentId) {
      this.currentAction = Action.Pull;
      this.cdr.detectChanges();
    }
  }

  async pushClick() {
    if (this.currentAction === Action.Push) return;

    this.replaceData("Pushing...");
    this.currentId = await Sync("push");
    if (this.currentId) {
      this.currentAction = Action.Push;
      this.cdr.detectChanges();
    }
  }

  async biClick() {
    if (this.currentAction === Action.Bi) return;

    this.replaceData("Bi...");
    this.currentId = await Sync("bi");
    if (this.currentId) {
      this.currentAction = Action.Bi;
      this.cdr.detectChanges();
    }
  }

  stopCommand() {
    if (!this.currentAction) return;
    StopCommand(this.currentId);
  }
}
