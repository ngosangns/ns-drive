import { Injectable, OnDestroy, OnInit } from "@angular/core";
import { BehaviorSubject } from "rxjs";
import { config, dto, models } from "../../wailsjs/go/models";
import {
  Sync,
  StopCommand,
  GetConfigInfo,
  UpdateProfiles,
  GetRemotes,
  DeleteRemote,
  AddRemote,
  StopAddingRemote,
} from "../../wailsjs/go/backend/App";
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
  BiResync = "bi-resync",
}

@Injectable({
  providedIn: "root",
})
export class AppService implements OnInit, OnDestroy {
  readonly currentId$ = new BehaviorSubject<number>(0);
  readonly currentAction$ = new BehaviorSubject<Action | undefined>(undefined);
  readonly data$ = new BehaviorSubject<string[]>([]);
  readonly configInfo$: BehaviorSubject<models.ConfigInfo>;
  readonly remotes$ = new BehaviorSubject<config.Remote[]>([]);

  constructor() {
    const configInfo = new models.ConfigInfo();
    configInfo.profiles = [];
    this.configInfo$ = new BehaviorSubject<models.ConfigInfo>(configInfo);

    EventsOn("tofe", (data: any) => {
      data = <CommandDTO>JSON.parse(data);

      switch (data.command) {
        case dto.Command.command_started:
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
    this.getRemotes();
  }

  async ngOnInit() {}

  ngOnDestroy() {}

  replaceData(str: string) {
    this.data$.next([str]);
  }

  async pull(profile: models.Profile) {
    if (this.currentAction$.value === Action.Pull) return;

    this.replaceData("Pulling...");
    this.currentId$.next(await Sync(<Action>"pull", profile));
    if (this.currentId$.value) this.currentAction$.next(Action.Pull);
  }

  async push(profile: models.Profile) {
    if (this.currentAction$.value === Action.Push) return;

    this.replaceData("Pushing...");
    this.currentId$.next(await Sync(<Action>"push", profile));
    if (this.currentId$.value) this.currentAction$.next(Action.Push);
  }

  async bi(profile: models.Profile, resync = false) {
    if (this.currentAction$.value === Action.Bi) return;

    this.replaceData("Bi...");
    this.currentId$.next(
      await Sync(resync ? <Action>"bi-resync" : <Action>"bi", profile)
    );
    if (this.currentId$.value) this.currentAction$.next(Action.Bi);
  }

  stopCommand() {
    if (!this.currentAction$.value) return;
    StopCommand(this.currentId$.value);
  }

  async getConfigInfo() {
    try {
      const configInfo = await GetConfigInfo();
      configInfo.profiles = configInfo.profiles ?? [];
      this.configInfo$.next(configInfo);
    } catch (e) {
      console.error(e);
      alert("Error getting config info");
    }
  }

  async getRemotes() {
    try {
      const remotes = await GetRemotes();
      this.remotes$.next(remotes ?? []);
    } catch (e) {
      console.error(e);
      alert("Error getting remotes");
    }
  }

  async addRemote(objData: Record<string, string>) {
    try {
      const err = await AddRemote(objData["name"], objData["type"], {});
      if (err) throw err.message;
      await this.getRemotes();
    } catch (e) {
      console.error(e);
    }
  }

  async stopAddingRemote() {
    try {
      const err = await StopAddingRemote();
      if (err) throw err.message;
    } catch (e) {
      console.error(e);
    }
  }

  async deleteRemote(name: string) {
    try {
      await DeleteRemote(name);
      await this.getRemotes();
    } catch (e) {
      console.error(e);
      alert("Error deleting remote");
    }
  }

  addProfile() {
    const profile = new models.Profile();
    profile.included_paths = [];
    profile.excluded_paths = [];
    this.configInfo$.value.profiles.push(profile);
    this.updateConfigInfo();
  }

  removeProfile(index: number) {
    this.configInfo$.value.profiles.splice(index, 1);
    this.updateConfigInfo();
  }

  updateConfigInfo() {
    this.configInfo$.next(this.configInfo$.value);
  }

  async saveConfigInfo() {
    try {
      const err = await UpdateProfiles(this.configInfo$.value.profiles);
      if (err) {
        throw err.message;
      }
    } catch (e) {
      console.error(e);
    }
  }

  addIncludePath(profileIndex: number) {
    this.configInfo$.value.profiles[profileIndex].included_paths.push("");
    this.updateConfigInfo();
  }

  removeIncludePath(profileIndex: number, index: number) {
    this.configInfo$.value.profiles[profileIndex].included_paths.splice(
      index,
      1
    );
    this.updateConfigInfo();
  }

  addExcludePath(profileIndex: number) {
    this.configInfo$.value.profiles[profileIndex].excluded_paths.push("");
    this.updateConfigInfo();
  }

  removeExcludePath(profileIndex: number, index: number) {
    this.configInfo$.value.profiles[profileIndex].excluded_paths.splice(
      index,
      1
    );
    this.updateConfigInfo();
  }
}
