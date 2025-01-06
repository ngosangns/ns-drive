import { Routes } from "@angular/router";
import { HomeComponent } from "./home/home.component";
import { SettingsComponent } from "./settings/settings.component";

export const routes: Routes = [
  {
    path: "",
    component: HomeComponent,
  },
  {
    path: "settings",
    component: SettingsComponent,
  },
];
