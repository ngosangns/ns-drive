import {
  ChangeDetectorRef,
  Injectable,
  OnDestroy,
  OnInit,
} from "@angular/core";
import { BehaviorSubject } from "rxjs";
import { dto, models } from "../../wailsjs/go/models";
import { Sync, StopCommand, GetConfigInfo } from "../../wailsjs/go/backend/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

interface CommandDTO {
  command: string;
  pid: number | undefined;
  task: string | undefined;
  error: string | undefined;
}

export enum Action {
  Pull = "pull",
  Push = "push",
  Bi = "bi",
}

@Injectable({
  providedIn: "root",
})
export class AppService implements OnInit, OnDestroy {
  readonly currentId$ = new BehaviorSubject<number>(0);
  readonly currentAction$ = new BehaviorSubject<Action | undefined>(undefined);
  readonly data$ = new BehaviorSubject<string[]>([]);
  readonly configInfo$ = new BehaviorSubject<models.ConfigInfo>(
    new models.ConfigInfo()
  );

  constructor() {
    EventsOn("tofe", (data: any) => {
      data = <CommandDTO>JSON.parse(data);

      switch (data.command) {
        case dto.Command.command_started:
          this;
          this.replaceData("Command started...");
          break;
        case dto.Command.command_stoped:
          this.currentAction$.next(undefined);
          this.currentId$.next(0);
          break;
        case dto.Command.command_output:
          this.replaceData(data.error);
          break;
        case dto.Command.error:
          const dataValue = this.data$.value;
          dataValue.push(data);
          this.data$.next(dataValue);
          break;
      }
    });

    this.getConfigInfo();
  }

  async ngOnInit() {}

  ngOnDestroy() {}

  replaceData(str: string) {
    this.data$.next([str]);
  }

  async pull() {
    if (this.currentAction$.value === Action.Pull) return;

    this.replaceData("Pulling...");
    this.currentId$.next(await Sync("pull"));
    if (this.currentId$.value) this.currentAction$.next(Action.Pull);
  }

  async push() {
    if (this.currentAction$.value === Action.Push) return;

    this.replaceData("Pushing...");
    this.currentId$.next(await Sync("push"));
    if (this.currentId$.value) this.currentAction$.next(Action.Push);
  }

  async bi() {
    if (this.currentAction$.value === Action.Bi) return;

    this.replaceData("Bi...");
    this.currentId$.next(await Sync("bi"));
    if (this.currentId$.value) this.currentAction$.next(Action.Bi);
  }

  stopCommand() {
    if (!this.currentAction$.value) return;
    StopCommand(this.currentId$.value);
  }

  async getConfigInfo() {
    try {
      const configInfo = await GetConfigInfo();
      this.configInfo$.next(configInfo);
    } catch (e) {
      console.error(e);
      alert("Error getting config info");
    }
  }
}
